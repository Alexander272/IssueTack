package tickets

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service services.Tickets
}

func NewHandler(service services.Tickets) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Tickets, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	tickets := api.Group("/tickets", middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Read()))
	{
		tickets.GET("", handlers.getAll)
		tickets.GET("/:id", handlers.getByID)
		tickets.GET("/:id/duplicates", handlers.getDuplicates)
		tickets.POST("/:id/take", handlers.take)
		tickets.POST("/:id/transfer", handlers.transfer)
		tickets.GET("/:id/due-date-requests", handlers.getDueDateRequests)
		tickets.POST("/:id/due-date-requests", handlers.createDueDateRequest)
		// Предпросмотр подкатегории — read-only, доступен всем создателям заявки
		// (в т.ч. безгрупповым, которые не могут выбрать подкатегорию вручную).
		tickets.POST("/modifier/preview", handlers.previewModifier)

		tickets.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Write()))
		tickets.POST("", handlers.create)
		tickets.PUT("/:id", handlers.update)
		tickets.POST("/:id/due-date-requests/:requestId/decide", handlers.decideDueDateRequest)
		tickets.POST("/:id/duplicate", handlers.linkDuplicate)
		tickets.POST("/:id/unlink-duplicate", handlers.unlinkDuplicate)

		tickets.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceTicket).Delete()))
		tickets.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	filter := &models.TicketFilter{}
	if err := c.ShouldBindQuery(&filter); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := normalizeTicketFilter(filter); err != nil {
		response.SendError(c, err)
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if ok {
		filter.RealmID = &realmID
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	filter.Actor = actor

	if favType := c.Query("favoritesType"); favType != "" {
		ft := models.FavoriteType(favType)
		if !ft.IsValid() {
			response.SendError(c, fmt.Errorf("%w: invalid favoritesType", models.ErrInvalidInput))
			return
		}
		filter.FavoritesByUser = &actor.ID
		filter.FavoriteType = &ft
	}

	data, total, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err, filter)
		return
	}
	response.SendData(c, data, total)
}

func normalizeTicketFilter(filter *models.TicketFilter) error {
	if len(filter.Statuses) > 0 {
		statuses := make([]models.TicketStatus, 0, len(filter.Statuses))
		for _, raw := range splitCSVValues(stringSliceFromStatuses(filter.Statuses)) {
			st := models.TicketStatus(raw)
			if !st.IsValid() {
				return fmt.Errorf("%w: invalid status %q", models.ErrInvalidInput, raw)
			}
			statuses = append(statuses, st)
		}
		filter.Statuses = statuses
	}

	if len(filter.Priorities) > 0 {
		priorities := make([]models.Priority, 0, len(filter.Priorities))
		for _, raw := range splitCSVValues(stringSliceFromPriorities(filter.Priorities)) {
			p := models.Priority(raw)
			if !p.IsValid() {
				return fmt.Errorf("%w: invalid priority %q", models.ErrInvalidInput, raw)
			}
			priorities = append(priorities, p)
		}
		filter.Priorities = priorities
	}

	if len(filter.SiteIDs) > 0 {
		filter.SiteIDs = splitCSVValues(filter.SiteIDs)
	}

	return nil
}

func splitCSVValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			out = append(out, part)
		}
	}
	return out
}

func stringSliceFromStatuses(values []models.TicketStatus) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func stringSliceFromPriorities(values []models.Priority) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

func (h *Handler) getByID(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)

	data, err := h.service.GetByID(c, &models.GetTicketByIdDTO{ID: id, Actor: actor, RealmID: realmIdStr})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

// getDuplicates отдаёт дубли основной заявки. Роут закрыт тем же read-правом
// ticket:read, что и детали заявки; атрибуты и каждого дубля проверяет сервис.
func (h *Handler) getDuplicates(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	data, err := h.service.GetDuplicates(c, &models.GetDuplicatesDTO{ID: id, Actor: actor})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) take(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)

	if err := h.service.Take(c, &models.TakeTicketDTO{ID: id, Actor: actor, RealmID: realmIdStr}); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Заявка взята в работу"})
}

func (h *Handler) transfer(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.TransferTicketDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = &id
	if dto.AssigneeID == nil || *dto.AssigneeID == uuid.Nil {
		response.SendError(c, models.ErrInvalidInput)
		return
	}

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)
	dto.RealmID = realmIdStr

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Transfer(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Заявка передана"})
}

func (h *Handler) getDueDateRequests(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	data, err := h.service.GetDueDateRequest(c, &models.GetDueDateRequestDTO{TicketID: id, Actor: actor})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) createDueDateRequest(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.CreateDueDateRequestDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.TicketID = id

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)
	dto.RealmID = realmIdStr

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	data, err := h.service.CreateDueDateRequest(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) decideDueDateRequest(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	requestId, err := uuid.Parse(c.Param("requestId"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.DecideDueDateRequestDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RequestID = requestId
	// :id — заявка из адреса: решение принимается только по её запросу, иначе
	// можно было бы утвердить перенос по чужой заявке, подставив её requestId.
	dto.TicketID = ticketID

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)
	dto.RealmID = realmIdStr

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	data, err := h.service.DecideDueDateRequest(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

// linkDuplicate помечает заявку из URL дублом основной, id которой пришёл в теле.
func (h *Handler) linkDuplicate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.LinkDuplicateDTO{}
	if err := c.ShouldBindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = id
	if dto.PrimaryID == uuid.Nil {
		response.SendError(c, models.ErrInvalidInput)
		return
	}
	dto.RealmID = utils.GetRealmString(c)

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	data, err := h.service.LinkDuplicate(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

// unlinkDuplicate снимает пометку дубля с заявки из URL.
func (h *Handler) unlinkDuplicate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	dto := &models.UnlinkDuplicateDTO{
		ID:      id,
		Actor:   actor,
		RealmID: utils.GetRealmString(c),
	}
	data, err := h.service.UnlinkDuplicate(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

// previewModifier — предпросмотр авто-подбора подкатегории для формы создания.
// Реалм только из заголовка: подбор идёт в границах реалма заявки, а не по
// значению из тела. Пустой результат — легальный ответ (null в data).
func (h *Handler) previewModifier(c *gin.Context) {
	dto := &models.ModifierPreviewDTO{}
	if err := c.ShouldBindJSON(dto); err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	dto.RealmID = realmID

	modifier, err := h.service.PreviewModifierByKeywords(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, modifier)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.TicketDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	dto.RealmID = &realmID

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Заявка создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.TicketDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if dto.ID == nil || id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	dto.RealmID = &realmID

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: dto.ID, Message: "Заявка обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	// Реалм берём из контекста (middleware проверил членство и права): строка
	// заголовка здесь — не авторизованное значение.
	realmIdStr := utils.GetRealmString(c)

	if err := h.service.Delete(c, &models.DeleteTicketDTO{ID: id, Actor: actor, RealmID: realmIdStr}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
