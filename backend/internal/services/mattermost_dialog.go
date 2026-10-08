package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

type recentTicket struct {
	id        uuid.UUID
	createdAt time.Time
}

// dialogCustomersWarnThreshold — с какого числа заказчиков пишем предупреждение
// в лог. Сам селект не ограничиваем: у диалогов Mattermost нет документированного
// предела на число опций, но очень длинный список на мобильных неудобен.
const dialogCustomersWarnThreshold = 300

// realmCustomers возвращает активных пользователей realm, не состоящих ни в одной
// группе, — это «заказчики» в терминах веб-формы создания заявки
// (membership=customers).
func (s *MattermostService) realmCustomers(ctx context.Context, realmID uuid.UUID) ([]*models.UserData, error) {
	users, err := s.users.GetByMembership(ctx, realmID, models.MembershipCustomers)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm customers: %w", err)
	}

	active := make([]*models.UserData, 0, len(users))
	for _, u := range users {
		if u.IsActive {
			active = append(active, u)
		}
	}
	return active, nil
}

// customerLabel собирает подпись заказчика для селекта в том же виде, что и веб-форма
// («Фамилия Имя (username)»), но без пустых скобок, если ФИО или username отсутствуют.
func customerLabel(u *models.UserData) string {
	name := strings.TrimSpace(u.LastName + " " + u.FirstName)
	switch {
	case name == "":
		return u.Username
	case u.Username == "" || u.Username == name:
		return name
	default:
		return fmt.Sprintf("%s (%s)", name, u.Username)
	}
}

// canPickOwner повторяет правило веб-формы: заказчика выбирают менеджеры
// (администраторы realm и управляющие группами) и исполнители (участники групп).
// Чистому заявителю поле не нужно — заявка его и так создаётся на себя.
func (s *MattermostService) canPickOwner(ctx context.Context, userID, realmID uuid.UUID) bool {
	isRealmAdmin := s.isRealmSupervisor(ctx, userID, realmID)

	managedGroups, err := s.groups.GetManagedGroups(ctx, userID, &realmID)
	if err != nil {
		logger.Warn("failed to get managed groups for owner picker",
			logger.StringAttr("user_id", userID.String()),
			logger.ErrAttr(err),
		)
	}
	memberGroups, err := s.groups.GetMemberGroups(ctx, userID, &realmID)
	if err != nil {
		logger.Warn("failed to get member groups for owner picker",
			logger.StringAttr("user_id", userID.String()),
			logger.ErrAttr(err),
		)
	}

	isManager := isRealmAdmin || len(managedGroups) > 0
	isExecutor := !isManager && len(memberGroups) > 0
	return isManager || isExecutor
}

// ownerSelectElement строит select «Заказчик» для диалога создания заявки. Поле
// доступно только тем, кто в веб-форме видит выбор заказчика (см. canPickOwner),
// и только если в realm есть хотя бы один активный заказчик: иначе возвращает
// nil и поле не показывается. Как и в веб-форме, поле обязательное; значение
// элемента — UUID пользователя, который на сабмите проверяется на принадлежность
// realm (см. dialogOwnerID).
func (s *MattermostService) ownerSelectElement(ctx context.Context, realmID uuid.UUID, mmUserID string) *mattermost.DialogElement {
	if mmUserID == "" {
		return nil
	}

	user, err := s.resolveOrCreateUser(ctx, realmID, mmUserID, nil)
	if err != nil {
		logger.Warn("failed to resolve user for owner picker",
			logger.StringAttr("realm_id", realmID.String()),
			logger.StringAttr("mm_user_id", mmUserID),
			logger.ErrAttr(err),
		)
		return nil
	}

	if !s.canPickOwner(ctx, user.ID, realmID) {
		return nil
	}

	customers, err := s.realmCustomers(ctx, realmID)
	if err != nil {
		logger.Warn("failed to load customers for owner picker",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return nil
	}

	if len(customers) == 0 {
		return nil
	}
	if len(customers) > dialogCustomersWarnThreshold {
		logger.Warn("large customer list in ticket dialog",
			logger.StringAttr("realm_id", realmID.String()),
			logger.Int32Attr("customers", int32(len(customers))),
		)
	}

	opts := make([]*model.PostActionOptions, 0, len(customers))
	for _, c := range customers {
		opts = append(opts, &model.PostActionOptions{Text: customerLabel(c), Value: c.ID.String()})
	}

	return &mattermost.DialogElement{
		DisplayName: "Заказчик",
		Name:        "ownerId",
		Type:        "select",
		Placeholder: "Выберите заказчика",
		Options:     opts,
	}
}

// dialogOwnerID разбирает значение поля ownerId из сабмита диалога. Значение
// приходит из клиента, поэтому проверяем всё, что проверяет ownerSelectElement:
// право нажать на выбор заказчика (canPickOwner) и то, что заказчик — активный
// участник realm вне групп. TicketService.Create членство в realm не проверяет
// (только OwnerID != nil), поэтому без этих проверок можно было бы подсунуть
// произвольного пользователя. Любая неудача не является ошибкой диалога: заявка
// просто уходит на создателя (owner = creator).
func (s *MattermostService) dialogOwnerID(ctx context.Context, realmID uuid.UUID, submitterID uuid.UUID, raw any) *uuid.UUID {
	rawID, ok := raw.(string)
	if !ok || rawID == "" {
		return nil
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		logger.Warn("dialog ownerId is not uuid",
			logger.StringAttr("realm_id", realmID.String()),
			logger.StringAttr("owner_raw", rawID),
		)
		return nil
	}

	if !s.canPickOwner(ctx, submitterID, realmID) {
		logger.Warn("dialog ownerId submitted by user without owner picker",
			logger.StringAttr("realm_id", realmID.String()),
			logger.StringAttr("user_id", submitterID.String()),
		)
		return nil
	}

	customers, err := s.realmCustomers(ctx, realmID)
	if err != nil {
		logger.Warn("failed to validate dialog ownerId",
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return nil
	}

	for _, c := range customers {
		if c.ID == id {
			return &id
		}
	}

	logger.Warn("dialog ownerId not in realm customers",
		logger.StringAttr("realm_id", realmID.String()),
		logger.StringAttr("owner_id", id.String()),
	)
	return nil
}

// HandleDialogOpen открывает диалог создания заявки для пользователя,
// нажавшего кнопку «Создать заявку». Подгружает категории/площадки realm
// и формирует поля диалога; в State передаёт канал и id кнопочного поста,
// чтобы после создания оповестить пользователя и удалить кнопку.
func (s *MattermostService) HandleDialogOpen(ctx context.Context, input *models.DialogOpenDTO) error {
	if input.TriggerID == "" {
		return fmt.Errorf("missing trigger_id")
	}
	if s.baseURL == "" {
		return fmt.Errorf("failed to open dialog: http.base_url is not configured")
	}

	realmID, err := uuid.Parse(input.Context["realm_id"])
	if err != nil {
		return fmt.Errorf("invalid realm_id: %w", err)
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return fmt.Errorf("failed to get mattermost settings: %w", err)
	}

	categories, err := s.categories.Get(ctx, &models.GetCategoriesDTO{RealmID: realmID})
	if err != nil {
		logger.Warn("failed to load categories for dialog", logger.ErrAttr(err))
	}

	sites, err := s.sites.Get(ctx, &models.GetSitesDTO{})
	if err != nil {
		logger.Warn("failed to load sites for dialog", logger.ErrAttr(err))
	}

	elements := []mattermost.DialogElement{
		{DisplayName: "Заголовок", Name: "title", Type: "text", MaxLength: 150},
		{DisplayName: "Описание", Name: "description", Type: "textarea", MaxLength: 3000},
	}

	if ownerEl := s.ownerSelectElement(ctx, realmID, input.UserID); ownerEl != nil {
		elements = append(elements, *ownerEl)
	}

	if len(categories) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(categories))
		for _, c := range categories {
			// У диалогов Mattermost нет группировки опций, поэтому раздел
			// попадает в подпись: «Раздел · Категория».
			text := c.Name
			if c.CategoryGroup != nil {
				text = fmt.Sprintf("%s · %s", c.CategoryGroup.Name, c.Name)
			}
			opts = append(opts, &model.PostActionOptions{Text: text, Value: c.ID.String()})
		}
		elements = append(elements, mattermost.DialogElement{
			DisplayName: "Категория", Name: "categoryId", Type: "select", Options: opts,
		})
	}

	if len(sites) > 0 {
		opts := make([]*model.PostActionOptions, 0, len(sites))
		for _, st := range sites {
			opts = append(opts, &model.PostActionOptions{Text: st.Name, Value: st.ID.String()})
		}
		elements = append(elements, mattermost.DialogElement{
			DisplayName: "Площадка", Name: "siteId", Type: "select", Options: opts,
		})
	}

	if err := s.most.Dialog.Open(settings.BotToken, mattermost.OpenRequest{
		TriggerID:   input.TriggerID,
		RealmID:     realmID.String(),
		Title:       "Новая заявка",
		SubmitLabel: "Создать",
		Elements:    elements,
		State:       fmt.Sprintf(`{"channelId":"%s","buttonPostId":"%s"}`, input.ChannelID, input.ButtonPostID),
	}); err != nil {
		return fmt.Errorf("failed to open dialog: %w", err)
	}
	return nil
}

// HandleDialogSubmission обрабатывает отправку диалога: создание заявки либо
// возврат заявки в работу (callback_id «reopen:<ticket_id>»). При создании
// резолвит пользователя Mattermost в систему, прикрепляет загруженные файлы и
// удаляет исходный кнопочный пост. Отмена диалога ничего не делает.
func (s *MattermostService) HandleDialogSubmission(ctx context.Context, submission *model.SubmitDialogRequest) error {
	if submission.Cancelled {
		return nil
	}

	// Не наш диалог создания заявки — callback_id несёт действие над заявкой.
	if action, ticketID, ok := parseTicketActionCallback(submission.CallbackId); ok {
		if action == actionReopen {
			return s.handleReopenSubmission(ctx, submission, ticketID)
		}
		return s.handleCancelSubmission(ctx, submission, action, ticketID)
	}

	realmID, err := uuid.Parse(submission.CallbackId)
	if err != nil {
		return fmt.Errorf("invalid callback_id: %w", err)
	}

	var (
		creator *models.UserData
		siteID  *uuid.UUID
		ownerID *uuid.UUID
	)
	if rawSite, ok := submission.Submission["siteId"].(string); ok && rawSite != "" {
		if id, err := uuid.Parse(rawSite); err == nil {
			siteID = &id
		}
	}

	creator, err = s.resolveOrCreateUser(ctx, realmID, submission.UserId, siteID)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	// ownerId приходит из диалога, поэтому значение можно подделать запросом: проверяем
	// и право выбирать заказчика (как в HandleDialogOpen), и членство в realm.
	ownerID = s.dialogOwnerID(ctx, realmID, creator.ID, submission.Submission["ownerId"])

	title, _ := submission.Submission["title"].(string)
	dto := &models.TicketDTO{
		Title:     title,
		Status:    models.StatusOpen,
		RealmID:   &realmID,
		CreatorID: creator.ID,
		Actor:     &models.Actor{ID: creator.ID, Name: creator.Username},
	}

	if ownerID != nil {
		dto.OwnerID = ownerID
	}

	if desc, ok := submission.Submission["description"].(string); ok && desc != "" {
		dto.Description = desc
	}
	if catID, ok := submission.Submission["categoryId"].(string); ok && catID != "" {
		id, err := uuid.Parse(catID)
		if err == nil {
			dto.CategoryID = id

			cat, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: id, RealmID: realmID})
			if err == nil {
				groupID := cat.GroupID
				dto.GroupID = &groupID
				if cat.Priority != "" {
					dto.Priority = cat.Priority
				}
			}
		}
	}

	if dto.GroupID != nil {
		group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID})
		if err == nil {
			if dto.AssigneeID == nil && group.DefaultAssigneeID != nil {
				dto.AssigneeID = group.DefaultAssigneeID
			}
			if dto.ManagerID == nil && group.ManagerID != nil {
				dto.ManagerID = group.ManagerID
			}
		}
	}
	if siteID != nil {
		dto.SiteID = *siteID
	}

	err = s.tickets.Create(ctx, dto)
	if err != nil {
		return fmt.Errorf("failed to create ticket: %w", err)
	}

	logger.Info("ticket created from mattermost",
		logger.StringAttr("ticket_id", dto.ID.String()),
		logger.StringAttr("mm_user", submission.UserId),
	)

	s.recentTickets.Store(submission.UserId+":"+submission.ChannelId, &recentTicket{
		id:        *dto.ID,
		createdAt: time.Now(),
	})

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err == nil {
		s.sendTicketCreatedDM(settings, submission.UserId, dto, creator)
		s.deleteButtonPost(settings, submission)
		s.processPendingFiles(ctx, settings.BotToken, submission, dto)
	}

	return nil
}

// HandleInteractiveAction обрабатывает нажатия интерактивных кнопок Mattermost:
// показывает список «Мои заявки» и меняет статус заявки по кнопке карточки.
// Возвращает ответ на нажатие (или nil, если ответ не нужен).
func (s *MattermostService) HandleInteractiveAction(ctx context.Context, input *models.InteractiveActionDTO) (*ActionResult, error) {
	if s.baseURL == "" {
		return nil, fmt.Errorf("failed to handle action: http.base_url is not configured")
	}

	action := input.Context["action"]

	switch action {
	case "my_tickets":
		// Кнопка «Мои заявки»: отвечаем тем же списком карточек, что и команда
		// «мои», но отдельным сообщением в диалоге — ответ action'а вида
		// {"update": post} заменил бы собой пост с кнопками, и меню пропало бы.
		if err := s.sendMyTickets(ctx, input); err != nil {
			return nil, err
		}
		return nil, nil

	case actionCancel, actionConfirm, actionReopen:
		return s.handleTicketStatusAction(ctx, action, input)

	default:
		return nil, nil
	}
}

// handleTicketStatusAction отвечает на кнопку действия в карточке заявки из
// списка «Мои заявки». Все три действия необратимы (cancelled и closed —
// терминальные статусы), поэтому бот не меняет статус сразу: сначала открывает
// диалог, а переход делает уже обработчик сабмита — handleCancelSubmission или
// handleReopenSubmission. Правила перехода применяет TicketService.Update, здесь
// только оркестрация: резолв реалма и открытие диалога.
func (s *MattermostService) handleTicketStatusAction(ctx context.Context, action string, input *models.InteractiveActionDTO) (*ActionResult, error) {
	ticketID, err := uuid.Parse(input.Context["ticket_id"])
	if err != nil {
		return nil, fmt.Errorf("invalid ticket_id: %w", err)
	}
	from, _ := strconv.Atoi(input.Context["from"])

	realmID, err := s.tickets.GetRealmIDByTicketID(ctx, ticketID)
	if err != nil {
		logger.Warn("mattermost action: failed to resolve ticket realm", logger.ErrAttr(err))
		return &ActionResult{Ephemeral: "Заявка не найдена"}, nil
	}
	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil || settings == nil {
		return nil, fmt.Errorf("failed to find realm settings: %w", err)
	}

	if action == actionReopen {
		return s.openReopenDialog(ctx, settings, ticketID, realmID, from, input)
	}
	return s.openCancelDialog(ctx, settings, action, ticketID, realmID, from, input)
}

// openCancelDialog просит подтвердить необратимый переход: отмену заявки или
// подтверждение её решения. Полей в диалоге нет — Mattermost сам показывает
// пару Submit/Cancel, поэтому подтверждение выглядит так же, как модалка в
// веб-плагине. Смещение среза и id сообщения едут в State: ответ диалога не
// умеет обновлять посты, карточку правит сам handleCancelSubmission.
func (s *MattermostService) openCancelDialog(ctx context.Context, settings *models.RealmMattermost, action string, ticketID, realmID uuid.UUID, from int, input *models.InteractiveActionDTO) (*ActionResult, error) {
	if input.TriggerID == "" {
		return &ActionResult{Ephemeral: "Не удалось открыть диалог: Mattermost не передал trigger_id"}, nil
	}

	req := mattermost.OpenRequest{
		TriggerID:   input.TriggerID,
		RealmID:     realmID.String(),
		Title:       "Отменить заявку",
		SubmitLabel: "Отменить заявку",
		State:       fmt.Sprintf(`{"from":%d,"post_id":%q}`, from, input.PostID),
	}
	switch action {
	case actionConfirm:
		req.CallbackID = confirmCallbackPrefix + ticketID.String()
		req.Title = "Подтвердить решение"
		req.SubmitLabel = "Закрыть заявку"
		req.Introduction = "Подтвердить решение и закрыть заявку? Действие необратимо."
	default:
		req.CallbackID = cancelCallbackPrefix + ticketID.String()
		req.Introduction = "Отменить заявку? Действие необратимо."
	}
	if subject := s.ticketSubject(ctx, ticketID); subject != "" {
		req.Introduction = subject + ". " + req.Introduction
	}

	if err := s.most.Dialog.Open(settings.BotToken, req); err != nil {
		logger.Warn("failed to open confirmation dialog",
			logger.StringAttr("action", action),
			logger.ErrAttr(err),
		)
		return &ActionResult{Ephemeral: "Не удалось открыть диалог"}, nil
	}
	return nil, nil
}

// ticketSubject — «Заявка №12. Не открывается 1С» для диалогов над заявкой:
// пользователь подтверждает необратимое действие и должен видеть, о какой
// заявке речь, даже если карточка уже уехала из экрана.
func (s *MattermostService) ticketSubject(ctx context.Context, ticketID uuid.UUID) string {
	ticket, err := s.tickets.GetSummary(ctx, ticketID)
	if err != nil || ticket == nil {
		return ""
	}
	switch {
	case ticket.TicketNumber != nil && cardSummary(ticket.Title) != "":
		return fmt.Sprintf("Заявка №%d. %s", *ticket.TicketNumber, cardSummary(ticket.Title))
	case ticket.TicketNumber != nil:
		return fmt.Sprintf("Заявка №%d", *ticket.TicketNumber)
	case cardSummary(ticket.Title) != "":
		return fmt.Sprintf("Заявка «%s»", cardSummary(ticket.Title))
	}
	return ""
}

// actionDoneText — подтверждение выполненного действия для нажавшего.
func actionDoneText(action string) string {
	if action == actionConfirm {
		return "Заявка закрыта."
	}
	return "Заявка отменена."
}

// changeTicketStatus переводит заявку в новый статус от имени пользователя
// Mattermost. Права на переход, терминальные статусы и прочие правила
// проверяет TicketService.Update.
func (s *MattermostService) changeTicketStatus(ctx context.Context, user *models.UserData, realmID, ticketID uuid.UUID, status models.TicketStatus) error {
	dto := &models.TicketDTO{
		ID:      &ticketID,
		Status:  status,
		Actor:   &models.Actor{ID: user.ID, Name: user.Username},
		RealmID: &realmID,
	}
	dto.MarkProvided("status")
	return s.tickets.Update(ctx, dto)
}

// openReopenDialog спрашивает причину возврата заявки в работу. Порядок как в
// плагине: сначала меняем статус, потом пишем комментарий с причиной — на
// «решённой» заявке внешний комментарий запрещён.
func (s *MattermostService) openReopenDialog(ctx context.Context, settings *models.RealmMattermost, ticketID, realmID uuid.UUID, from int, input *models.InteractiveActionDTO) (*ActionResult, error) {
	if input.TriggerID == "" {
		return &ActionResult{Ephemeral: "Не удалось открыть диалог: Mattermost не передал trigger_id"}, nil
	}

	intro := "Укажите причину возврата заявки в работу."
	if subject := s.ticketSubject(ctx, ticketID); subject != "" {
		intro = subject + ". " + intro
	}

	err := s.most.Dialog.Open(settings.BotToken, mattermost.OpenRequest{
		TriggerID:    input.TriggerID,
		RealmID:      realmID.String(),
		CallbackID:   reopenCallbackPrefix + ticketID.String(),
		Title:        "Вернуть заявку в работу",
		Introduction: intro,
		SubmitLabel:  "Вернуть в работу",
		Elements: []mattermost.DialogElement{{
			DisplayName: "Причина",
			Name:        "reason",
			Type:        "textarea",
			MaxLength:   1000,
			Placeholder: "Например: причина обращения не устранена",
		}},
		// post_id кладём в State: в SubmitDialogRequest его нет, а после
		// возврата в работу карточку в списке надо перерисовать.
		State: fmt.Sprintf(`{"from":%d,"post_id":%q}`, from, input.PostID),
	})
	if err != nil {
		logger.Warn("failed to open reopen dialog", logger.ErrAttr(err))
		return &ActionResult{Ephemeral: "Не удалось открыть диалог"}, nil
	}
	return nil, nil
}

// handleReopenSubmission обрабатывает отправку диалога «Вернуть в работу»:
// переводит заявку в работу и сохраняет причину комментарием от владельца.
func (s *MattermostService) handleReopenSubmission(ctx context.Context, submission *model.SubmitDialogRequest, ticketID uuid.UUID) error {
	reason, _ := submission.Submission["reason"].(string)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return models.ErrReasonRequired
	}

	realmID, err := s.tickets.GetRealmIDByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to resolve ticket realm: %w", err)
	}
	user, err := s.resolveOrCreateUser(ctx, realmID, submission.UserId, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	if err := s.changeTicketStatus(ctx, user, realmID, ticketID, models.StatusInProgress); err != nil {
		return fmt.Errorf("failed to reopen ticket: %w", err)
	}

	// Комментарий с причиной — отдельным шагом: если он не прошёл, заявка всё
	// равно уже в работе, и молчаливую потерю причины лучше залогировать.
	if _, err := s.comments.Create(ctx, nil, &models.CreateCommentDTO{
		Text:     "Заявка возвращена в работу: " + reason,
		TicketID: ticketID,
		UserID:   user.ID,
		Realm:    realmID.String(),
	}); err != nil {
		logger.Warn("failed to save reopen reason comment",
			logger.StringAttr("ticket_id", ticketID.String()),
			logger.ErrAttr(err),
		)
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil || settings == nil || settings.BotToken == "" {
		return nil
	}

	msg := "Заявка возвращена в работу. Причина: " + reason
	if _, err := s.most.Post.Create(settings.BotToken, mattermost.CreatePostDTO{
		ChannelID: submission.ChannelId,
		Message:   msg,
	}); err != nil {
		logger.Warn("failed to send reopen confirmation", logger.ErrAttr(err))
	}

	// Заявка осталась в списке активных: обновляем её карточку на месте —
	// снятые после возврата кнопки с «решённого» статуса больше не нажмут.
	state := parseDialogState(submission.State)
	if err := s.applyListEdit(ctx, settings, realmID, user, state, false); err != nil {
		logger.Warn("failed to update list post after reopen",
			logger.StringAttr("post_id", state.PostID),
			logger.ErrAttr(err),
		)
	}
	return nil
}

// handleCancelSubmission обрабатывает подтверждение необратимого перехода:
// отмены заявки (ticket_cancel) или подтверждения её решения (ticket_confirm).
// Полей в этом диалоге нет, поэтому кроме состояния диалога проверять нечего:
// меняем статус и убираем карточку из списка.
func (s *MattermostService) handleCancelSubmission(ctx context.Context, submission *model.SubmitDialogRequest, action string, ticketID uuid.UUID) error {
	realmID, err := s.tickets.GetRealmIDByTicketID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("failed to resolve ticket realm: %w", err)
	}
	user, err := s.resolveOrCreateUser(ctx, realmID, submission.UserId, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve user: %w", err)
	}

	// Отказ сервиса тикетов (заявку уже закрыли, сняли права и т.п.) показываем
	// в диалоге: карточку в ленте не трогаем, список обновится при следующем
	// «Мои заявки».
	if err := s.changeTicketStatus(ctx, user, realmID, ticketID, actionTargetStatus(action)); err != nil {
		logger.Info("mattermost ticket action rejected",
			logger.StringAttr("action", action),
			logger.StringAttr("ticket_id", ticketID.String()),
			logger.ErrAttr(err),
		)
		return err
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil || settings == nil || settings.BotToken == "" {
		return nil
	}

	// Отменённая и закрытая заявка уходят из активного списка: карточка
	// убирается из своего сообщения.
	state := parseDialogState(submission.State)
	if err := s.applyListEdit(ctx, settings, realmID, user, state, true); err != nil {
		logger.Warn("failed to update list post after cancel",
			logger.StringAttr("action", action),
			logger.StringAttr("post_id", state.PostID),
			logger.ErrAttr(err),
		)
	}

	// Ответ диалога не умеет отправлять сообщения нажавшему, поэтому итог
	// подтверждения уходит обычным постом в канал — как при возврате в работу.
	if _, err := s.most.Post.Create(settings.BotToken, mattermost.CreatePostDTO{
		ChannelID: submission.ChannelId,
		Message:   actionDoneText(action),
	}); err != nil {
		logger.Warn("failed to send cancel confirmation",
			logger.StringAttr("action", action),
			logger.ErrAttr(err),
		)
	}
	return nil
}

// applyListEdit приводит сообщение списка «Мои заявки» в соответствие с новым
// составом заявок пользователя. Общий путь для сабмитов диалогов: ответ диалога
// не умеет обновлять посты (SubmitDialogResponse знает только error), поэтому
// карточку правим сами.
//
// removed=true — заявка ушла из активных (отмена, подтверждение решения):
// карточка убирается из своего сообщения, соседи остаются на местах, ничего из
// следующего сообщения не подтягивается; если карточка была единственной,
// сообщение удаляется. removed=false — заявка осталась в списке (возврат в
// работу): срез перерисовывается целиком, карточка получает новый статус.
func (s *MattermostService) applyListEdit(ctx context.Context, settings *models.RealmMattermost, realmID uuid.UUID, user *models.UserData, state dialogState, removed bool) error {
	if state.PostID == "" {
		// Диалог мог быть открыт до того, как бот начал класть post_id в State.
		return nil
	}

	tickets, err := s.myActiveTickets(ctx, realmID, user)
	if err != nil {
		return fmt.Errorf("failed to reload user tickets: %w", err)
	}

	chunk, ok := s.myTicketsChunkAfterEdit(tickets, state.From, removed, user)
	if !ok {
		// Сообщение опустело только когда карточка была в нём единственной,
		// то есть смещение ровно совпало с новым концом списка. Смещение больше
		// конца списка — это битое значение в State, удалять по нему пост нельзя:
		// показываем начало списка.
		if state.From == len(tickets) {
			return s.most.Post.Delete(settings.BotToken, state.PostID)
		}
		chunk, ok = s.myTicketsChunkAt(tickets, 0, user)
		if !ok {
			return nil
		}
	}
	return s.most.Post.UpdateCards(settings.BotToken, state.PostID, chunk.message, chunk.cards)
}

// dialogState — то, что бот кладёт в State диалога: смещение среза списка и id
// сообщения, из которого диалог открыт.
type dialogState struct {
	From   int    `json:"from"`
	PostID string `json:"post_id"`
}

func parseDialogState(raw string) dialogState {
	var state dialogState
	if raw == "" {
		return state
	}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		logger.Warn("invalid dialog state", logger.StringAttr("state", raw), logger.ErrAttr(err))
		return dialogState{}
	}
	return state
}

// parseTicketActionCallback разбирает callback_id диалогов над заявкой:
// «ticket_cancel:<id>», «ticket_confirm:<id>», «ticket_reopen:<id>».
func parseTicketActionCallback(callbackID string) (string, uuid.UUID, bool) {
	action, raw, ok := strings.Cut(callbackID, ":")
	if !ok {
		return "", uuid.Nil, false
	}
	switch action {
	case actionCancel, actionConfirm, actionReopen:
	default:
		return "", uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", uuid.Nil, false
	}
	return action, id, true
}

// actionErrorMessage переводит ошибку сервиса в текст для нажавшего кнопку:
// доменные ошибки уже несут человеческий текст, остальное — общий fallback.
func actionErrorMessage(err error) string {
	var domainErr *models.DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Message()
	}
	return "Не удалось изменить статус заявки"
}

// sendMyTickets отвечает на нажатие «Мои заявки»: находит realm по context,
// разрешает пользователя Mattermost в ApplicationUser и отправляет в диалог
// карточки его активных заявок.
func (s *MattermostService) sendMyTickets(ctx context.Context, input *models.InteractiveActionDTO) error {
	realmID, err := uuid.Parse(input.Context["realm_id"])
	if err != nil {
		return fmt.Errorf("failed to send my tickets: invalid realm_id: %w", err)
	}

	settings, err := s.repo.GetByRealm(ctx, realmID)
	if err != nil {
		return fmt.Errorf("failed to find realm settings: %w", err)
	}
	if settings == nil {
		return fmt.Errorf("failed to send my tickets: mattermost is not configured for realm %s", realmID)
	}

	ch := &mmChannel{
		Settings:  settings,
		MmUserID:  input.UserID,
		ChannelID: input.ChannelID,
	}
	return s.sendStatusMessage(ctx, ch)
}

// sendTicketCreatedDM отправляет пользователю личное сообщение с подтверждением
// создания заявки и ссылкой на неё. Ошибки отправки не фатальны — заявка уже
// создана, поэтому проблема лишь логируется.
func (s *MattermostService) sendTicketCreatedDM(settings *models.RealmMattermost, mmUserID string, dto *models.TicketDTO, creator *models.UserData) {
	if settings.BotToken == "" {
		return
	}
	msg := fmt.Sprintf("Заявка №%d создана.\nЗаголовок: %s",
		dto.TicketNumber, dto.Title)
	if s.showWebLink(creator, dto.RealmID) {
		msg += fmt.Sprintf("\nОткрыть: %s/tasks/%s", s.baseURL, dto.ID.String())
	} else {
		msg += fmt.Sprintf("\n[Открыть в плагине](%s)", pluginDeepLink(*dto.ID))
	}
	msg += "\nОтправьте файлы следующим сообщением в течение 30 минут — они прикрепятся автоматически к этой заявке. Или можете указать номер заявки (например, №123) вместе с файлами."
	if err := s.most.DM.Send(settings.BotToken, settings.BotUserID, mmUserID, msg); err != nil {
		logger.Warn("failed to send ticket created DM", logger.ErrAttr(err))
	}
}

// deleteButtonPost удаляет исходный пост с кнопкой «Создать заявку» после того,
// как диалог был успешно отправлен. Состояние (id поста кнопки) передаётся
// в диалог заранее через State; если id отсутствует, пост не трогаем.
func (s *MattermostService) deleteButtonPost(settings *models.RealmMattermost, submission *model.SubmitDialogRequest) {
	if settings.BotToken == "" || submission.State == "" {
		return
	}
	var state struct {
		ButtonPostID string `json:"buttonPostId"`
	}
	if err := json.Unmarshal([]byte(submission.State), &state); err != nil || state.ButtonPostID == "" {
		return
	}
	if err := s.most.Post.Delete(settings.BotToken, state.ButtonPostID); err != nil {
		logger.Warn("failed to delete button post", logger.ErrAttr(err))
	}
}

// processPendingFiles прикрепляет файлы, загруженные пользователем вместе с
// сообщением-командой, к только что созданной заявке. Ключ — канал+пользователь,
// т.к. Mattermost не связывает файлы загруженные до диалога напрямую. Каждый
// файл скачивается отдельно, ошибки не прерывают обработку остальных.
func (s *MattermostService) processPendingFiles(ctx context.Context, botToken string, submission *model.SubmitDialogRequest, dto *models.TicketDTO) {
	key := submission.ChannelId + ":" + submission.UserId
	fileIDsVal, ok := s.pendingFiles.LoadAndDelete(key)
	if !ok {
		logger.Debug("no pending files found", logger.StringAttr("key", key))
		return
	}
	fileIDs, ok := fileIDsVal.([]string)
	if !ok || len(fileIDs) == 0 {
		return
	}

	logger.Info("processing pending files",
		logger.StringAttr("ticket_id", dto.ID.String()),
		logger.IntAttr("count", len(fileIDs)),
	)

	for _, fileID := range fileIDs {
		data, err := s.most.Client.DownloadFile(botToken, fileID)
		if err != nil {
			logger.Warn("failed to download MM file",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}

		info, err := s.most.Client.GetFileInfo(botToken, fileID)
		if err != nil {
			logger.Warn("failed to get MM file info",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}

		fileName := info.Name
		if fileName == "" {
			fileName = fileID
		}

		att, err := s.tickets.UploadAttachment(ctx, nil, &models.UploadAttachmentDTO{
			EntityType: "ticket",
			EntityID:   *dto.ID,
			FileName:   fileName,
			FileSize:   info.Size,
			MimeType:   info.MimeType,
			File:       bytes.NewReader(data),
			UploadedBy: dto.CreatorID,
			Realm:      dto.RealmID.String(),
		})
		if err != nil {
			logger.Warn("failed to upload MM file as attachment",
				logger.StringAttr("file_id", fileID),
				logger.ErrAttr(err),
			)
			continue
		}
		logger.Info("attachment created from MM file",
			logger.StringAttr("ticket_id", dto.ID.String()),
			logger.StringAttr("attachment_id", att.ID.String()),
			logger.StringAttr("file_name", fileName),
		)
	}
}
