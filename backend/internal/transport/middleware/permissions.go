package middleware

import (
	"errors"
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// checkRealmMembership проверяет, что пользователь состоит в реалме, домен которого
// пришёл из клиента. Заголовок realm задаёт Casbin-домен, поэтому без этой проверки
// пользователь мог подставить чужой realm и получить coarse-права (ticket:read/write)
// в чужой области: тикеты грузятся по id без realm-фильтра, а доступ к уже загруженному
// тикету считается по ticket.RealmID (TicketAccessService.CheckAccessOnTicket).
//
// Пустой realm не проверяем — на нём Enforce всё равно ничего не разрешит, и такое
// поведение нужно для маршрутов вне реалмов (например, собственный профиль).
func (m *Middleware) checkRealmMembership(c *gin.Context, userID uuid.UUID, realmId string) error {
	if realmId == "" {
		return nil
	}

	realmID, err := uuid.Parse(realmId)
	if err != nil {
		// Не-UUID не может быть валидным Casbin-доменом, но явный отказ понятнее,
		// чем «нет прав» из-за молчаливо не найденного реалма.
		return models.ErrPermissionDenied
	}

	membership, err := m.services.UserRealms.GetByUserAndRealm(c, userID, realmID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.ErrPermissionDenied
		}
		return err
	}
	if membership == nil || !membership.IsActive {
		return models.ErrPermissionDenied
	}

	return nil
}

func (m *Middleware) CheckPermissions(perms ...access.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		u, exists := c.Get(constants.CtxUser)
		if !exists {
			response.SendError(c, models.ErrSessionEmpty)
			return
		}
		user, ok := u.(models.User)
		if !ok {
			response.SendError(c, fmt.Errorf("invalid user type in context"))
			return
		}

		realmId := c.GetHeader("realm")
		if realmId == "" {
			realmId = c.Query("realm")
		}
		if err := m.checkRealmMembership(c, user.ID, realmId); err != nil {
			response.SendError(c, err)
			return
		}

		var accessAllowed bool
		var lastErr error

		for _, r := range perms {
			ok, err := m.services.AccessPolicies.Enforce(user.ID.String(), realmId, string(r.Resource), string(r.Action))
			if err != nil {
				lastErr = err
				continue
			}
			if ok {
				accessAllowed = true
				break
			}
		}

		if lastErr != nil && !accessAllowed {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrPolicyCheck, lastErr))
			return
		}

		if !accessAllowed {
			response.SendError(c, models.ErrPermissionDenied)
			return
		}

		c.Next()
	}
}
