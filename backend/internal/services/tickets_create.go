package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

// autoAssign при создании тикета достраивает поля из группы:
//   - исполнитель, если не указан явно: приоритет — ответственный по умолчанию группы
//     (DefaultAssigneeID), иначе — единственный участник группы, чтобы тикет не остался без
//     исполнителя. Если в группе несколько участников, исполнитель не назначается.
//   - менеджер (ManagerID) — менеджером группы (ManagerID), если не задан явно.
//
// Группа ищется с предиктом realm тикета: group_id приходит из тела, поэтому без
// него можно было бы привязать заявку к группе чужого реалма — её участники
// получили бы доступ к заявке, а ответственный по умолчанию стал бы исполнителем
// из другого реалма. Чужой id отвечает ErrNotFound, как и несуществующий.
func (s *TicketService) autoAssign(ctx context.Context, dto *models.TicketDTO) error {
	if dto.RealmID == nil {
		return fmt.Errorf("realm is required")
	}

	group, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID, RealmID: dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get group: %w", err)
	}

	if dto.AssigneeID == nil {
		if group.DefaultAssigneeID != nil {
			dto.AssigneeID = group.DefaultAssigneeID
		} else {
			count, err := s.groups.GetMemberCount(ctx, *dto.GroupID)
			if err != nil {
				return fmt.Errorf("failed to get member count: %w", err)
			}
			if count == 1 {
				members, err := s.groups.GetMembers(ctx, &models.GetGroupDTO{ID: *dto.GroupID, RealmID: dto.RealmID})
				if err != nil {
					return fmt.Errorf("failed to get members: %w", err)
				}
				if len(members) > 0 {
					dto.AssigneeID = &members[0].ID
				}
			}
		}
	}

	if dto.ManagerID == nil && group.ManagerID != nil {
		dto.ManagerID = group.ManagerID
	}
	return nil
}

// checkGroupRealm ограничивает смену группы рамками реалма тикета.
//
// Группа принадлежит реалму, поэтому перенос заявки в группу соседнего отдела
// отдал бы её участникам доступ к заявке (участие в группе даёт read, менеджер —
// write/delete), а ответственному по умолчанию — ещё и назначение исполнителем.
// Право на смену группы есть только у администратора реалма, но сама группа
// проверяется по realm тикета, а не по значению из тела запроса. Чужой id
// отвечает ErrNotFound.
//
// Снятие группы (groupId = null) не проверяется: реалм при этом не меняется.
func (s *TicketService) checkGroupRealm(ctx context.Context, oldTicket *models.Ticket, dto *models.TicketDTO) error {
	if dto.GroupID == nil {
		return nil
	}
	if oldTicket.RealmID == nil {
		return fmt.Errorf("ticket realm is required")
	}
	if _, err := s.groups.GetByID(ctx, &models.GetGroupDTO{ID: *dto.GroupID, RealmID: oldTicket.RealmID}); err != nil {
		return fmt.Errorf("failed to check group realm: %w", err)
	}
	return nil
}

// Create создаёт новый тикет: проверяет права на создание, заполняет
// владельца/исполнителя (при необходимости — автоматически), фиксирует
// действие в журнале и отправляет уведомление.
func (s *TicketService) Create(ctx context.Context, dto *models.TicketDTO) error {
	if dto.Actor == nil {
		return models.ErrPermissionDenied
	}
	if strings.TrimSpace(dto.Title) == "" {
		return models.ErrInvalidInput
	}

	realmStr := ""
	if dto.RealmID != nil {
		realmStr = dto.RealmID.String()
	}

	ok, err := s.access.CanCreateTicket(ctx, dto.Actor.ID, realmStr)
	if err != nil {
		return fmt.Errorf("policy check failed: %w", err)
	}

	if !ok {
		// Исполнитель (участник хотя бы одной группы) может создать заявку
		// с ограничениями: приоритет/группа/исполнитель принудительно
		// определяются системой. Заказчик необязателен: если он не указан,
		// ниже им становится создатель.
		member, err := s.groups.GetMemberGroups(ctx, dto.Actor.ID, dto.RealmID)
		if err != nil {
			return fmt.Errorf("failed to get member groups: %w", err)
		}
		if len(member) == 0 {
			return models.ErrPermissionDenied
		}
		if err := s.applyExecutorCreateRestrictions(ctx, dto, member); err != nil {
			return err
		}
	}

	// Группа обязательна. Если не передана явно, берём её из категории —
	// так же, как это делает плагин Mattermost (PluginCreateTicket) и
	// путь исполнителя (applyExecutorCreateRestrictions). У категории
	// группа определена всегда, поэтому явный выбор нужен только там,
	// где его хочет сделать менеджер; пустое значение = «по категории».
	if dto.GroupID == nil || *dto.GroupID == uuid.Nil {
		if dto.CategoryID == uuid.Nil || dto.RealmID == nil {
			return fmt.Errorf("group is required: %w", models.ErrInvalidInput)
		}
		category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: *dto.RealmID})
		if err != nil {
			return fmt.Errorf("failed to get category for group: %w", err)
		}
		if category.GroupID == uuid.Nil {
			return fmt.Errorf("category has no group")
		}
		dto.GroupID = &category.GroupID
		if !dto.Priority.IsValid() {
			dto.Priority = category.Priority
		}
	}

	// Заказчик необязателен: если он не указан, заявка выполняется для создателя.
	// Подстановка обязательна — колонка tickets.owner_id в БД NOT NULL, поэтому
	// пустой заказчик из формы не должен уходить в INSERT как NULL.
	// Серверная часть правила выбора заказчика (P2): явный заказчик должен быть
	// выбран разрешённым пользователем (менеджер/исполнитель) и быть активным
	// пользователем области — иначе заявка с чужим заказчиком не создаётся.
	explicitOtherOwner := dto.OwnerID != nil && *dto.OwnerID != dto.CreatorID
	if dto.OwnerID != nil {
		if dto.RealmID == nil {
			return models.ErrInvalidInput
		}
		if err := s.validateOwnerSelection(ctx, dto.Actor.ID, *dto.RealmID, *dto.OwnerID); err != nil {
			return err
		}
	}
	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if err := s.autoAssign(ctx, dto); err != nil {
		return fmt.Errorf("auto-assign: %w", err)
	}

	canSetDueDate := false
	isExecutor := false

	isManager, err := s.isManagerOfGroup(ctx, &models.GroupShort{ID: *dto.GroupID}, dto.Actor.ID)
	if err != nil {
		return fmt.Errorf("failed to check manager access: %w", err)
	}

	supervisor, err := s.access.IsRealmSupervisor(ctx, dto.Actor.ID, realmStr)
	if err != nil {
		return fmt.Errorf("failed to check supervisor access: %w", err)
	}

	if dto.OwnerMuted {
		if !isManager && !supervisor {
			return models.ErrPermissionDenied
		}
		if !explicitOtherOwner {
			dto.OwnerMuted = false
		}
	}

	// Исполнитель — участник хотя бы одной группы реалма (а не «тот, кто выбран
	// исполнителем этой заявки»): право на подкатегорию от назначения не зависит.
	// Запрос нужен и для срока, и для подкатегории; у менеджера/супервизора он
	// делается только при выбранной подкатегории — для срока они уже прошли.
	modifierSelected := dto.CategoryModifierID != nil && *dto.CategoryModifierID != uuid.Nil
	if (!isManager && !supervisor) || modifierSelected {
		memberGroups, err := s.groups.GetMemberGroups(ctx, dto.Actor.ID, dto.RealmID)
		if err != nil {
			return fmt.Errorf("failed to get member groups: %w", err)
		}
		isExecutor = len(memberGroups) > 0
	}
	canSetDueDate = isManager || supervisor || isExecutor

	// 1. Если пользователь пытается вручную установить срок, но не имеет прав — блокируем
	if dto.DueDate != nil && !canSetDueDate {
		return models.ErrPermissionDenied
	}

	// Подкатегорию при создании выбирает исполнитель (участник группы) или
	// начальник области — независимо от того, кто станет исполнителем заявки.
	if err := s.prepareCreateModifier(ctx, &createModifierInput{
		DTO: dto, Supervisor: supervisor, IsExecutor: isExecutor,
	}); err != nil {
		return err
	}

	// 2. Автосрок из категории применяется всем (свойство категории, а не право
	// пользователя). canSetDueDate гейтит только ручную установку срока.
	if dto.DueDate == nil && dto.CategoryID != uuid.Nil {
		category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: *dto.RealmID})
		if err != nil {
			return fmt.Errorf("failed to get category: %w", err)
		}
		if category.DefaultDueDays != nil && *category.DefaultDueDays > 0 {
			dd := time.Now().UTC().AddDate(0, 0, *category.DefaultDueDays)
			dto.DueDate = &dd
		}
	}

	if err := s.validateAndPrepareNotifiers(ctx, dto, nil); err != nil {
		return err
	}

	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
		}

		if dto.HasField("notifierIds") {
			if err := s.repo.SyncNotifiers(ctx, newTx, *dto.ID, dto.Actor.ID, dto.NotifierIDs); err != nil {
				return fmt.Errorf("failed to sync ticket notifiers: %w", err)
			}
		}

		if len(dto.Subtasks) > 0 {
			for _, st := range dto.Subtasks {
				st.TicketID = *dto.ID
				st.Actor = dto.Actor
			}
			if err := s.subtasks.CreateManyOnCreate(ctx, newTx, dto.Subtasks); err != nil {
				return fmt.Errorf("failed to create subtasks: %w", err)
			}
		}

		log := &models.ActivityLogDTO{
			Action:        "created",
			ChangedBy:     dto.Actor.ID,
			ChangedByName: dto.Actor.Name,
			EntityType:    string(access.ResourceTicket),
			EntityID:      *dto.ID,
			Entity:        dto.Title,
		}
		if err := log.SetNewValues(map[string]string{"title": dto.Title}); err != nil {
			return fmt.Errorf("set new values: %w", err)
		}
		if err := s.logs.Create(ctx, newTx, []*models.ActivityLogDTO{log}); err != nil {
			return fmt.Errorf("store log: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	created, err := s.repo.GetByID(ctx, &models.GetTicketByIdDTO{ID: *dto.ID})
	if err != nil {
		return fmt.Errorf("failed to load created ticket for notification: %w", err)
	}
	if err := s.notifications.TicketCreated(ctx, created, dto.Actor.ID); err != nil {
		s.notifyBestEffort("create", *dto.ID, dto.Title, err)
	}
	return nil
}

// applyExecutorCreateRestrictions нормализует и валидирует поля заявки, создаваемой
// исполнителем (участником группы, без coarse-прав CanCreateTicket). Исполнителю
// разрешено выбирать исполнителя (любого активного исполнителя реалма), приоритет,
// группу и срок; при отсутствии явного выбора применяются дефолты категории/группы.
func (s *TicketService) applyExecutorCreateRestrictions(ctx context.Context, dto *models.TicketDTO, memberGroups []uuid.UUID) error {
	if dto.RealmID == nil {
		return fmt.Errorf("realm is required")
	}

	category, err := s.categories.GetByID(ctx, &models.GetCategoryByIdDTO{ID: dto.CategoryID, RealmID: *dto.RealmID})
	if err != nil {
		return fmt.Errorf("failed to get category: %w", err)
	}
	if category.GroupID == uuid.Nil {
		return fmt.Errorf("category has no group")
	}

	// Группа: исполнитель может выбрать любую группу реалма; по умолчанию — группа категории.
	if dto.GroupID == nil || *dto.GroupID == uuid.Nil {
		dto.GroupID = &category.GroupID
	}

	// Приоритет: принимает выбранное валидное значение, иначе — приоритет категории.
	if !dto.Priority.IsValid() {
		dto.Priority = category.Priority
	}

	// Исполнитель: можно назначить любого активного исполнителя реалма (участника
	// хотя бы одной группы в реалме). Невалидный выбор отклоняется.
	if dto.AssigneeID != nil {
		if err := s.validateRealmExecutor(ctx, *dto.AssigneeID, *dto.RealmID); err != nil {
			return err
		}
	} else if containsUUID(memberGroups, category.GroupID) {
		// Категория ведёт в группу исполнителя и явного выбора нет — назначаем его.
		dto.AssigneeID = &dto.CreatorID
	}
	return nil
}

// validateRealmExecutor проверяет, что пользователь — активный исполнитель области:
// активен, состоит в области и является участником хотя бы одной группы области.
func (s *TicketService) validateRealmExecutor(ctx context.Context, userID, realmID uuid.UUID) error {
	if userID == uuid.Nil {
		return models.ErrInvalidInput
	}
	u, err := s.users.GetByID(ctx, userID)
	if err != nil || u == nil || !u.IsActive {
		return models.ErrInvalidInput
	}
	if !hasActiveRealmMembership(u, realmID) {
		return models.ErrInvalidInput
	}
	memberGroups, err := s.groups.GetMemberGroups(ctx, userID, &realmID)
	if err != nil {
		return fmt.Errorf("failed to get member groups: %w", err)
	}
	if len(memberGroups) == 0 {
		return models.ErrInvalidInput
	}
	return nil
}

// containsUUID сообщает, есть ли target в values.
func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
