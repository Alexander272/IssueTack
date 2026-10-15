package services

import (
	"context"
	"fmt"
	"strings"

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
		// с ограничениями: обязателен заказчик, а приоритет/группа/исполнитель/срок
		// принудительно определяются системой.
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

	if dto.GroupID == nil {
		return fmt.Errorf("group is required")
	}

	if dto.OwnerID == nil {
		dto.OwnerID = &dto.CreatorID
	}

	if err := s.autoAssign(ctx, dto); err != nil {
		return fmt.Errorf("auto-assign: %w", err)
	}

	// Срок выполнения при создании может ставить только менеджер группы тикета
	// или админ реалма (realm supervisor). Политика Casbin write права на срок
	// не даёт — участник с realm-wide write не может выставить dueDate.
	if dto.DueDate != nil {
		isManager, err := s.isManagerOfGroup(ctx, &models.GroupShort{ID: *dto.GroupID}, dto.Actor.ID)
		if err != nil {
			return fmt.Errorf("failed to check manager access: %w", err)
		}
		supervisor, err := s.access.IsRealmSupervisor(ctx, dto.Actor.ID, realmStr)
		if err != nil {
			return fmt.Errorf("failed to check supervisor access: %w", err)
		}
		if !isManager && !supervisor {
			return models.ErrPermissionDenied
		}
	}

	err = s.tx.WithinTransaction(ctx, func(newTx postgres.Tx) error {
		if err := s.repo.Create(ctx, newTx, dto); err != nil {
			return fmt.Errorf("failed to create ticket. error: %w", err)
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

// applyExecutorCreateRestrictions применяет ограничения при создании заявки исполнителем
// (участником группы, не имеющим полных прав). Заказчик обязателен; группа и приоритет
// принудительно берутся из категории, срок отсутствует. Исполнителем исполнитель может
// назначить только себя (явно — на любую группу, включая чужую; чужой id из тела
// отбрасывается). Если категория принадлежит одной из групп исполнителя, заявку и без
// явного выбора назначают на него, иначе исполнитель подбирается системой по умолчанию.
func (s *TicketService) applyExecutorCreateRestrictions(ctx context.Context, dto *models.TicketDTO, memberGroups []uuid.UUID) error {
	if dto.OwnerID == nil {
		return models.ErrOwnerRequired
	}
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

	dto.GroupID = &category.GroupID
	dto.Priority = category.Priority
	dto.DueDate = nil

	// Исполнитель может назначить исполнителем только себя — на любую группу, в т.ч. чужую
	// (берёт заявку на себя). Чужой id из тела доверять нельзя: без этой отбраковки он
	// прошёл бы в autoAssign и подставился бы ответственным группы уже чужой.
	if dto.AssigneeID != nil && *dto.AssigneeID != dto.CreatorID {
		dto.AssigneeID = nil
	}
	if containsUUID(memberGroups, category.GroupID) && dto.AssigneeID == nil {
		dto.AssigneeID = &dto.CreatorID
	}
	return nil
}

func containsUUID(values []uuid.UUID, target uuid.UUID) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
