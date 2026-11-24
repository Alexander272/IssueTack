package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// duplicateActionAllowedStatuses — обычные статусы, из которых можно пометить
// заявку дублем. closed/cancelled здесь исключены: дубль закрывается самой
// операцией link. Исключение для уже привязанного (закрытого) дубля делает
// duplicateSourceAllowed — иначе перенаправление было бы недостижимо.
func duplicateActionAllowedStatuses(status models.TicketStatus) bool {
	switch status {
	case models.StatusOpen, models.StatusInProgress, models.StatusPending, models.StatusOnHold, models.StatusResolved:
		return true
	}
	return false
}

// duplicateSourceAllowed пускает в link активные заявки и уже привязанные дубли.
// Произвольная закрытая заявка остаётся запрещённой: её нельзя «оживить» снятием
// связи, а у дубля для этого есть previous_status.
func duplicateSourceAllowed(ticket *models.Ticket) bool {
	if duplicateActionAllowedStatuses(ticket.Status) {
		return true
	}
	return ticket.Status == models.StatusClosed && ticket.DuplicateOf != nil
}

// duplicatePrimaryAllowed — основную можно выбрать в любом статусе, кроме
// отменённой. Решённая и закрытая допустимы; закрытый дубль как основная
// сплющивается в корень цепочки.
func duplicatePrimaryAllowed(ticket *models.Ticket) bool {
	return ticket.Status != models.StatusCancelled
}

func ticketRef(t *models.Ticket) string {
	if t == nil {
		return "none"
	}
	if t.TicketNumber != nil {
		return fmt.Sprintf("№%d", *t.TicketNumber)
	}
	return t.ID.String()
}

// duplicateUserLabel — короткая подпись человека в карточке дубля. Пустой
// (незаполненный) UserShort не должен превращаться в «  » в ответе API.
func duplicateUserLabel(user *models.UserShort) string {
	if user == nil || user.ID == uuid.Nil {
		return ""
	}
	name := strings.TrimSpace(user.LastName + " " + user.FirstName)
	if name == "" {
		name = strings.TrimSpace(user.Username)
	}
	return name
}

// canWorkDuplicate — рабочий доступ к дублю: исполнитель/менеджер/начальник области.
// Автор-заказчик без групп сюда не попадает (IsCustomer), так как решение о дубле
// принимает тот, кто ведёт заявку. Намеренно не используем CheckWorkAccess: он
// запрещает любые рабочие операции на resolved/closed, а resolved-заявку
// разрешено связывать (иначе исполнитель потерял бы эту возможность).
func (s *TicketService) canWorkDuplicate(ctx context.Context, ticket *models.Ticket, userID uuid.UUID) (bool, error) {
	err := s.access.CheckAccessOnTicket(ctx, ticket, userID, string(access.Write))
	if err != nil && !errors.Is(err, models.ErrPermissionDenied) {
		return false, err
	}
	worker := err == nil || (ticket.Assignee != nil && ticket.Assignee.ID == userID)
	if !worker {
		return s.access.CanManage(ctx, userID, ticket)
	}
	customer, err := s.access.IsCustomer(ctx, ticket, userID)
	if err != nil {
		return false, err
	}
	return !customer, nil
}

// canUnlinkDuplicate — та же формула, по которой UnlinkDuplicate принимает решение.
// Используется и при отдаче списка дублей, чтобы кнопка «Отвязать» показывалась
// только там, где сервис действительно разрешит операцию.
func (s *TicketService) canUnlinkDuplicate(ctx context.Context, duplicate, primary *models.Ticket, userID uuid.UUID) (bool, error) {
	worker, err := s.canWorkDuplicate(ctx, duplicate, userID)
	if err != nil {
		return false, err
	}
	if worker {
		return true, nil
	}
	manager, err := s.access.CanManage(ctx, userID, primary)
	if err != nil {
		return false, err
	}
	if manager {
		return true, nil
	}
	return duplicate.Creator.ID == userID, nil
}

// duplicateAccess проверяет доступ к паре «дубль — основная». На основную
// требуется read-доступ: ссылку на чужую заявку пользователь не должен ни
// увидеть, ни получить наблюдателя через неё.
func (s *TicketService) duplicateAccess(ctx context.Context, duplicate, primary *models.Ticket, userID uuid.UUID) error {
	allowed, err := s.canWorkDuplicate(ctx, duplicate, userID)
	if err != nil {
		return err
	}
	if !allowed {
		return models.ErrPermissionDenied
	}
	if primary != nil {
		if !sameRealm(duplicate, primary) {
			return models.ErrInvalidInput
		}
		if err := s.access.CheckAccessOnTicket(ctx, primary, userID, string(access.Read)); err != nil {
			return err
		}
	}
	return nil
}

func sameRealm(a, b *models.Ticket) bool {
	if a.RealmID == nil || b.RealmID == nil {
		return a.RealmID == nil && b.RealmID == nil
	}
	return *a.RealmID == *b.RealmID
}

// GetDuplicates отдаёт дубли основной заявки. Рекурсивный обход тут тот же, что
// при связывании: он ограничен глубиной duplicateChainLimit и не зацикливается.
// Каждая найденная заявка дополнительно проходит CheckAccessOnTicket — список
// не должен раскрывать дубли, которые актору не положено видеть.
func (s *TicketService) GetDuplicates(ctx context.Context, dto *models.GetDuplicatesDTO) ([]*models.DuplicateRef, error) {
	if dto == nil || dto.Actor == nil {
		return nil, models.ErrInvalidInput
	}

	primary, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
	if err != nil {
		return nil, err
	}
	if err := s.access.CheckAccessOnTicket(ctx, primary, dto.Actor.ID, string(access.Read)); err != nil {
		return nil, err
	}

	ids, err := s.repo.GetDuplicateDescendants(ctx, nil, []uuid.UUID{primary.ID})
	if err != nil {
		return nil, err
	}

	// Дубли всегда closed, поэтому статус в карточке бесполезен. А объём
	// комментариев и вложений — наоборот, показывает, что там обсуждалось.
	activity, err := s.repo.CountDuplicateActivity(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to count duplicate activity: %w", err)
	}

	refs := make([]*models.DuplicateRef, 0, len(ids))
	for _, id := range ids {
		if id == primary.ID {
			continue
		}
		ticket, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: id})
		if err != nil {
			return nil, fmt.Errorf("failed to load duplicate %s: %w", id, err)
		}
		if err := s.access.CheckAccessOnTicket(ctx, ticket, dto.Actor.ID, string(access.Read)); err != nil {
			if errors.Is(err, models.ErrPermissionDenied) {
				continue
			}
			return nil, err
		}
		canUnlink, err := s.canUnlinkDuplicate(ctx, ticket, primary, dto.Actor.ID)
		if err != nil {
			return nil, err
		}
		counts := activity[ticket.ID]
		refs = append(refs, &models.DuplicateRef{
			ID:               ticket.ID,
			TicketNumber:     ticket.TicketNumber,
			Title:            ticket.Title,
			CreatorName:      duplicateUserLabel(&ticket.Creator),
			AssigneeName:     duplicateUserLabel(ticket.Assignee),
			CommentsCount:    counts.Comments,
			AttachmentsCount: counts.Attachments,
			CreatedAt:        ticket.CreatedAt,
			CanUnlink:        canUnlink,
		})
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].TicketNumber != nil && refs[j].TicketNumber != nil && *refs[i].TicketNumber != *refs[j].TicketNumber {
			return *refs[i].TicketNumber < *refs[j].TicketNumber
		}
		if refs[i].TicketNumber != nil && refs[j].TicketNumber == nil {
			return true
		}
		if refs[i].TicketNumber == nil && refs[j].TicketNumber != nil {
			return false
		}
		return refs[i].Title < refs[j].Title
	})

	return refs, nil
}

// LinkDuplicate связывает заявку с основной. Если основная сама является дублем,
// цепочка сплющивается: все ссылки переводятся на её корень. Операция целиком
// идёт в одной транзакции, с блокировкой всей затронутой ветки.
func (s *TicketService) LinkDuplicate(ctx context.Context, dto *models.LinkDuplicateDTO) (*models.DuplicateLinkResult, error) {
	if dto == nil || dto.Actor == nil {
		return nil, models.ErrInvalidInput
	}
	if dto.ID == dto.PrimaryID {
		return nil, models.ErrDuplicateOwnTicket
	}

	var result *models.DuplicateLinkResult
	err := s.tx.WithinTransaction(ctx, func(tx postgres.Tx) error {
		duplicate, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}
		primary, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.PrimaryID})
		if err != nil {
			return models.ErrDuplicateNotFound
		}
		if err := s.duplicateAccess(ctx, duplicate, primary, dto.Actor.ID); err != nil {
			return err
		}
		if !duplicateSourceAllowed(duplicate) {
			return models.ErrDuplicateStatusNotAllowed
		}
		if !duplicatePrimaryAllowed(primary) {
			return models.ErrDuplicatePrimaryInactive
		}

		// Блокируем все затронутые вершины до пересчёта цепочки: параллельные
		// link/unlink не должны оставить петлю или «потерять» потомков.
		ids := []uuid.UUID{duplicate.ID, primary.ID}
		for _, start := range []uuid.UUID{duplicate.ID, primary.ID} {
			chain, err := s.repo.GetDuplicateChain(ctx, tx, start)
			if err != nil {
				return err
			}
			for _, node := range chain {
				ids = append(ids, node.ID)
			}
		}
		if err := s.lockDuplicateTree(ctx, tx, ids); err != nil {
			return err
		}

		// После блокировки состояние могло измениться — перечитываем и проверяем
		// заново, чтобы гонка не превратила цепочку в цикл.
		duplicate, err = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}
		primary, err = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.PrimaryID})
		if err != nil {
			return models.ErrDuplicateNotFound
		}
		if !duplicateSourceAllowed(duplicate) {
			return models.ErrDuplicateStatusNotAllowed
		}
		if !duplicatePrimaryAllowed(primary) {
			return models.ErrDuplicatePrimaryInactive
		}

		chain, err := s.repo.GetDuplicateChain(ctx, tx, primary.ID)
		if err != nil {
			return err
		}
		for _, node := range chain {
			if node.ID == duplicate.ID {
				return models.ErrDuplicateCycle
			}
		}
		root := chain[len(chain)-1].ID
		result = &models.DuplicateLinkResult{RootID: root, Chain: chainIDs(append([]postgres.DuplicateNode{{ID: duplicate.ID}}, chain...))}

		// Основная сама дубль → переносим всех её дублей на корень, чтобы
		// цепочка не углублялась и её устойчивость не зависела от порядка операций.
		if root != primary.ID {
			children, err := s.repo.GetDuplicateDescendants(ctx, tx, []uuid.UUID{primary.ID})
			if err != nil {
				return err
			}
			if err := s.repo.RepointDuplicates(ctx, tx, primary.ID, root); err != nil {
				return err
			}
			if err := s.repo.MoveDuplicateNotifiers(ctx, tx, primary.ID, root); err != nil {
				return err
			}
			var repointed []uuid.UUID
			for _, id := range children {
				if id != primary.ID && id != duplicate.ID {
					repointed = append(repointed, id)
				}
			}
			if err := s.writeDuplicateLogs(ctx, tx, dto.Actor, models.ActionDuplicateRepointed, repointed, ticketRef(primary), root.String()); err != nil {
				return err
			}
			primary, err = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: root})
			if err != nil {
				return err
			}
		}
		result.RootTicketNumber = primary.TicketNumber

		previousStatus := duplicate.Status
		oldPrimaryID := duplicate.DuplicateOf
		switch {
		case oldPrimaryID == nil:
			// Новая связь: закрываем дубль и запоминаем прежний статус.
			if err := s.repo.SetDuplicate(ctx, tx, duplicate.ID, root, previousStatus); err != nil {
				return err
			}
			if err := s.favorites.DeleteTemporaryByTickets(ctx, tx, []uuid.UUID{duplicate.ID}); err != nil {
				return fmt.Errorf("failed to cleanup favorites of linked duplicate: %w", err)
			}
			if err := s.writeDuplicateLogs(ctx, tx, dto.Actor, models.ActionDuplicateLinked,
				[]uuid.UUID{duplicate.ID}, "none", ticketRef(primary)); err != nil {
				return err
			}
			if err := s.writeStatusLog(ctx, tx, dto.Actor, duplicate, previousStatus, models.StatusClosed); err != nil {
				return err
			}
		case *oldPrimaryID != root:
			// Дубль уже привязан: меняем только основную, status и previous_status
			// остаются как есть (заявка уже закрыта).
			if err := s.repo.RepointOne(ctx, tx, duplicate.ID, root); err != nil {
				return err
			}
			if err := s.repo.MoveDuplicateNotifier(ctx, tx, *oldPrimaryID, root, duplicate.ID); err != nil {
				return err
			}
			if err := s.writeDuplicateLogs(ctx, tx, dto.Actor, models.ActionDuplicateRepointed,
				[]uuid.UUID{duplicate.ID}, oldPrimaryID.String(), ticketRef(primary)); err != nil {
				return err
			}
		default:
			// Выбрали основную, на которую дубль уже ссылается (например, после
			// flatten промежуточную основу). Переписывать и логировать нечего —
			// observers ниже всё равно остаются привязанными к корню.
		}

		// Автор, заказчик и явные наблюдатели дубля становятся неявными
		// наблюдателями основной — иначе ссылка из уведомления будет для них
		// неоткрываемой. Актора НЕ исключаем: если он сам автор дубля, ему всё
		// равно нужен доступ к основной. От уведомления о собственном действии
		// его отсекает NotifyDuplicate.
		for _, userID := range duplicateObserverIDs(duplicate) {
			if err := s.repo.AddDuplicateNotifier(ctx, tx, root, userID, duplicate.ID, dto.Actor.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifyDuplicateBestEffort(ctx, dto.ID, result.RootID, dto.Actor.ID, true)
	return result, nil
}

// UnlinkDuplicate снимает пометку дубля. Это осознанное исключение из
// ErrTicketFrozen: статус восстанавливается из previous_status, closed_at
// сбрасывается, resolved_at сохраняется.
func (s *TicketService) UnlinkDuplicate(ctx context.Context, dto *models.UnlinkDuplicateDTO) (*models.DuplicateLinkResult, error) {
	if dto == nil || dto.Actor == nil {
		return nil, models.ErrInvalidInput
	}

	var result *models.DuplicateLinkResult
	err := s.tx.WithinTransaction(ctx, func(tx postgres.Tx) error {
		duplicate, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}
		if duplicate.DuplicateOf == nil {
			return models.ErrDuplicateNotLinked
		}
		primary, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *duplicate.DuplicateOf})
		if err != nil {
			return err
		}
		allowed, err := s.canUnlinkDuplicate(ctx, duplicate, primary, dto.Actor.ID)
		if err != nil {
			return err
		}
		if !allowed {
			return models.ErrPermissionDenied
		}

		ids := []uuid.UUID{duplicate.ID, primary.ID}
		chain, err := s.repo.GetDuplicateChain(ctx, tx, primary.ID)
		if err != nil {
			return err
		}
		for _, node := range chain {
			ids = append(ids, node.ID)
		}
		if err := s.lockDuplicateTree(ctx, tx, ids); err != nil {
			return err
		}

		duplicate, err = s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: dto.ID})
		if err != nil {
			return err
		}
		if duplicate.DuplicateOf == nil {
			return models.ErrDuplicateNotLinked
		}
		oldPrimaryID := *duplicate.DuplicateOf
		status := duplicate.Status
		restored := models.StatusOpen
		if duplicate.PreviousStatus != nil {
			restored = *duplicate.PreviousStatus
		}

		if err := s.repo.ClearDuplicate(ctx, tx, duplicate.ID); err != nil {
			return err
		}
		if err := s.repo.DeleteDuplicateNotifiers(ctx, tx, duplicate.ID); err != nil {
			return err
		}
		if err := s.writeDuplicateLogs(ctx, tx, dto.Actor, models.ActionDuplicateUnlinked,
			[]uuid.UUID{duplicate.ID}, oldPrimaryID.String(), "none"); err != nil {
			return err
		}
		if err := s.writeStatusLog(ctx, tx, dto.Actor, duplicate, status, restored); err != nil {
			return err
		}
		result = &models.DuplicateLinkResult{
			RootID:           oldPrimaryID,
			RootTicketNumber: primary.TicketNumber,
			Chain:            []uuid.UUID{duplicate.ID, oldPrimaryID},
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.notifyDuplicateBestEffort(ctx, dto.ID, result.RootID, dto.Actor.ID, false)
	return result, nil
}

// lockDuplicateTree блокирует вершины подветки и пересобирает снимок уже под
// блокировкой. Параллельный link может добавить потомка между первым обходом
// и захватом локов; такой потомок тоже попадает под лок, иначе граф (а с ним
// repoint и журнал операций) считался бы по устаревшему снимку.
func (s *TicketService) lockDuplicateTree(ctx context.Context, tx postgres.Tx, roots []uuid.UUID) error {
	locked := uniqueUUIDs(roots)
	if len(locked) == 0 {
		return nil
	}
	if err := s.repo.LockTickets(ctx, tx, locked); err != nil {
		return err
	}

	for {
		descendants, err := s.repo.GetDuplicateDescendants(ctx, tx, locked)
		if err != nil {
			return err
		}
		known := make(map[uuid.UUID]struct{}, len(locked))
		for _, id := range locked {
			known[id] = struct{}{}
		}
		fresh := make([]uuid.UUID, 0, len(descendants))
		for _, id := range descendants {
			if _, ok := known[id]; !ok {
				fresh = append(fresh, id)
			}
		}
		if len(fresh) == 0 {
			return nil
		}
		if err := s.repo.LockTickets(ctx, tx, fresh); err != nil {
			return err
		}
		locked = append(locked, fresh...)
	}
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func chainIDs(nodes []postgres.DuplicateNode) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

// writeDuplicateLogs пишет журнал операции по всем затронутым заявкам одной
// записью на заявку — в той же транзакции, что и изменение связей.
func (s *TicketService) writeDuplicateLogs(
	ctx context.Context, tx postgres.Tx, actor *models.Actor,
	action models.ActivityType, ids []uuid.UUID, oldVal, newVal string,
) error {
	// В журнал пишем человекочитаемые ссылки: activityMeta.ts рендерит эти
	// значения как plain, и сырой uuid выглядел бы в «Истории» как мусор.
	// В Data уведомлений остаются исходные id — там их парсит нотификатор.
	oldDisplay := s.resolveTicketRefLabel(ctx, oldVal)
	newDisplay := s.resolveTicketRefLabel(ctx, newVal)
	logs := make([]*models.ActivityLogDTO, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		log := &models.ActivityLogDTO{
			Action:        string(action),
			ChangedBy:     actor.ID,
			ChangedByName: actor.Name,
			EntityType:    models.EntityTypeTicket,
			EntityID:      id,
			Entity:        ticketRef(&models.Ticket{ID: id}),
		}
		if err := log.SetOldValues(map[string]string{string(action): oldDisplay}); err != nil {
			return err
		}
		if err := log.SetNewValues(map[string]string{string(action): newDisplay}); err != nil {
			return err
		}
		logs = append(logs, log)
	}
	if len(logs) == 0 {
		return nil
	}
	if err := s.logs.Create(ctx, tx, logs); err != nil {
		return fmt.Errorf("store duplicate logs: %w", err)
	}
	return nil
}

// writeStatusLog фиксирует закрытие/восстановление статуса отдельной строкой:
// вкладка «История» показывает и саму связь дубля, и смену статуса.
func (s *TicketService) writeStatusLog(
	ctx context.Context, tx postgres.Tx, actor *models.Actor,
	ticket *models.Ticket, from, to models.TicketStatus,
) error {
	log := &models.ActivityLogDTO{
		Action:        string(models.ActionStatusChanged),
		ChangedBy:     actor.ID,
		ChangedByName: actor.Name,
		EntityType:    models.EntityTypeTicket,
		EntityID:      ticket.ID,
		Entity:        ticket.Title,
	}
	if err := log.SetOldValues(map[string]string{string(models.ActionStatusChanged): string(from)}); err != nil {
		return err
	}
	if err := log.SetNewValues(map[string]string{string(models.ActionStatusChanged): string(to)}); err != nil {
		return err
	}
	return s.logs.Create(ctx, tx, []*models.ActivityLogDTO{log})
}

// notifyDuplicateBestEffort — уведомление после коммита: недоступность
// Mattermost не превращает уже зафиксированную операцию в ошибку клиенту.
func (s *TicketService) notifyDuplicateBestEffort(ctx context.Context, duplicateID, primaryID, actorID uuid.UUID, linked bool) {
	duplicate, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: duplicateID})
	if err != nil {
		s.notifyBestEffort("duplicate", duplicateID, "", err)
		return
	}
	primary, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: primaryID})
	if err != nil {
		s.notifyBestEffort("duplicate", duplicateID, duplicate.Title, err)
		return
	}
	if err := s.notifications.NotifyDuplicate(ctx, duplicate, primary, actorID, linked); err != nil {
		s.notifyBestEffort("duplicate", duplicateID, duplicate.Title, err)
	}
}

// duplicateObserverIDs — все, кто должен следить за основной после связывания:
// автор и заказчик дубля плюс его явный список «Уведомлять». Источник явного
// списка (ticket_notifiers дубля) не трогается: при отвязке DeleteDuplicateNotifiers
// убирает только эти производные строки, а наблюдатели остаются на дубле.
// Это не список получателей события: актор исключается только при рассылке
// уведомления (NotifyDuplicate), но не здесь.
func duplicateObserverIDs(t *models.Ticket) []uuid.UUID {
	ids := []uuid.UUID{}
	seen := make(map[uuid.UUID]struct{})
	add := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	add(t.Creator.ID)
	if t.Owner != nil {
		add(t.Owner.ID)
	}
	for i := range t.Notifiers {
		add(t.Notifiers[i].ID)
	}
	return ids
}
