package statistics

import (
	"fmt"
	"strings"
	"time"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// defaultRangeDays — окно статистики по умолчанию, если период не задан.
const defaultRangeDays = 30

type Handler struct {
	service services.Statistics
}

func NewHandler(service services.Statistics) *Handler {
	return &Handler{service: service}
}

func Register(api *gin.RouterGroup, service services.Statistics, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	statistics := api.Group("/statistics", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		statistics.GET("/tickets", handlers.getTickets)
	}
}

type statisticsQuery struct {
	From        string `form:"from"`
	To          string `form:"to"`
	Granularity string `form:"granularity" binding:"omitempty,oneof=day week month"`
}

func (h *Handler) getTickets(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	query := &statisticsQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	from, to, err := parseRange(query.From, query.To)
	if err != nil {
		response.SendError(c, err)
		return
	}

	assigneeIDs, err := parseUUIDList(c, "assigneeId")
	if err != nil {
		response.SendError(c, err)
		return
	}
	categoryIDs, err := parseUUIDList(c, "categoryId")
	if err != nil {
		response.SendError(c, err)
		return
	}
	groupIDs, err := parseUUIDList(c, "groupId")
	if err != nil {
		response.SendError(c, err)
		return
	}
	siteIDs, err := parseUUIDList(c, "siteId")
	if err != nil {
		response.SendError(c, err)
		return
	}

	filter := &models.StatisticsFilter{
		Actor:       actor,
		RealmID:     realmID,
		From:        from,
		To:          to,
		Granularity: query.Granularity,
		AssigneeIDs: assigneeIDs,
		CategoryIDs: categoryIDs,
		GroupIDs:    groupIDs,
		SiteIDs:     siteIDs,
	}

	data, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

// parseRange разбирает период запроса. Даты — в формате YYYY-MM-DD; верхняя граница
// включительна для пользователя, поэтому to переводится в эксклюзивный конец суток.
// Пустой `to` — текущий момент, пустой `from` — начало дня за defaultRangeDays до `to`.
func parseRange(fromStr, toStr string) (time.Time, time.Time, error) {
	now := time.Now()

	from, to := time.Time{}, now
	if toStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", toStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid to", models.ErrInvalidInput)
		}
		to = parsed.AddDate(0, 0, 1)
	}
	if fromStr != "" {
		parsed, err := time.ParseInLocation("2006-01-02", fromStr, time.Local)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%w: invalid from", models.ErrInvalidInput)
		}
		from = parsed
	} else {
		from = to.AddDate(0, 0, -defaultRangeDays)
	}

	if !from.Before(to) {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: from must be before to", models.ErrInvalidInput)
	}

	return from, to, nil
}

// parseUUIDList читает список uuid из query-параметра. RTK Query сериализует массивы
// через запятую (`assigneeId=a,b`), но поддерживаются и повторяющиеся параметры
// (`assigneeId=a&assigneeId=b`), поэтому каждый элемент дополнительно режется по запятой.
// Пустые значения пропускаются; невалидный uuid — ошибка ввода.
func parseUUIDList(c *gin.Context, key string) ([]uuid.UUID, error) {
	raw := c.QueryArray(key)
	if len(raw) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(raw))
	for _, value := range raw {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := uuid.Parse(part)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid %s", models.ErrInvalidInput, key)
			}
			ids = append(ids, id)
		}
	}
	return ids, nil
}
