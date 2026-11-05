package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/google/uuid"
)

// CategoryGroupsLookup — узкая выборка групп, нужная только для решения о
// видимости скрытых категорий (её реализует GroupService).
type CategoryGroupsLookup interface {
	GetManagedGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
	GetMemberGroups(ctx context.Context, userID uuid.UUID, realmID *uuid.UUID) ([]uuid.UUID, error)
}

// CategoryVisibility решает, кому видны скрытые категории и разделы категорий.
// Правило — «менеджер или исполнитель» (зеркало MattermostService.canPickOwner):
// начальник области (realm supervisor), управляющий группой или участник группы.
// Безгрупповой заказчик («обычный пользователь») скрытые сущности не видит.
type CategoryVisibility struct {
	access TicketAccessChecker
	groups CategoryGroupsLookup
}

// NewCategoryVisibility создаёт CategoryVisibility.
func NewCategoryVisibility(access TicketAccessChecker, groups CategoryGroupsLookup) *CategoryVisibility {
	return &CategoryVisibility{access: access, groups: groups}
}

// CanSeeHidden сообщает, видит ли пользователь скрытые категории/разделы.
// Проверка best-effort: при сбое зависимостей возвращает false, то есть скрытое
// остаётся скрытым (fail-closed), а не роняет список категорий.
func (v *CategoryVisibility) CanSeeHidden(ctx context.Context, userID, realmID uuid.UUID) bool {
	if userID == uuid.Nil || realmID == uuid.Nil {
		return false
	}

	ok, err := v.access.IsRealmSupervisor(ctx, userID, realmID.String())
	if err != nil {
		logger.Warn("failed to check realm supervisor for hidden categories",
			logger.StringAttr("user_id", userID.String()),
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return false
	}
	if ok {
		return true
	}

	managed, err := v.groups.GetManagedGroups(ctx, userID, &realmID)
	if err != nil {
		logger.Warn("failed to get managed groups for hidden categories",
			logger.StringAttr("user_id", userID.String()),
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return false
	}
	if len(managed) > 0 {
		return true
	}

	member, err := v.groups.GetMemberGroups(ctx, userID, &realmID)
	if err != nil {
		logger.Warn("failed to get member groups for hidden categories",
			logger.StringAttr("user_id", userID.String()),
			logger.StringAttr("realm_id", realmID.String()),
			logger.ErrAttr(err),
		)
		return false
	}
	return len(member) > 0
}
