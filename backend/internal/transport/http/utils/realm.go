package utils

import (
	"fmt"

	"github.com/Alexander272/IssueTrack/backend/internal/constants"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// realmFromRequest достаёт реалм, под которым запрос авторизован.
//
// Основной источник — constants.CtxRealm, который кладёт CheckPermissions после
// успешной проверки членства и прав. Это важно: в Enforce уходит именно это
// значение, поэтому подставлять его в запросы к репозиторию нужно то же самое,
// а не то, что клиент прислал в заголовке (middleware принимает ещё и ?realm=).
// Заголовок остаётся запасным вариантом для роутов без CheckPermissions.
//
// Второй результат — найден ли realm вообще, третий — ошибка разбора.
func realmFromRequest(c *gin.Context) (uuid.UUID, bool, error) {
	s := ""
	if v, ok := c.Get(constants.CtxRealm); ok {
		if str, ok := v.(string); ok {
			s = str
		}
	}
	if s == "" {
		s = c.GetHeader("realm")
	}
	if s == "" {
		return uuid.Nil, false, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("%w: invalid realm header: %v", models.ErrInvalidInput, err)
	}
	return id, true, nil
}

// GetRealmUUID возвращает realm запроса, если он есть. Отсутствие realm не
// считается ошибкой — используется на роутах, где реалм необязателен.
func GetRealmUUID(c *gin.Context) (uuid.UUID, bool) {
	id, found, err := realmFromRequest(c)
	if err != nil {
		response.SendError(c, err)
		return uuid.Nil, false
	}
	return id, found
}

// GetRealmString — строковое представление авторизованного реалма для DTO, где
// поле хранится как string. Пустая строка означает «реалм не задан».
func GetRealmString(c *gin.Context) string {
	id, ok := GetRealmUUID(c)
	if !ok {
		return ""
	}
	return id.String()
}

// RequireRealmUUID — то же, что GetRealmUUID, но для роутов, где реалм обязателен:
// при отсутствии realm отвечает ошибкой. Возврат без ответа оставлял бы запрос
// висеть (gin ничего не пишет, клиент ждёт таймаут).
func RequireRealmUUID(c *gin.Context) (uuid.UUID, bool) {
	id, found, err := realmFromRequest(c)
	if err == nil && !found {
		err = fmt.Errorf("%w: не указан realm", models.ErrInvalidInput)
	}
	if err != nil {
		response.SendError(c, err)
		return uuid.Nil, false
	}
	return id, true
}
