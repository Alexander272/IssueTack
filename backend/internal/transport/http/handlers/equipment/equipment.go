package equipment

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Хендлеры оборудования разнесены по тематическим файлам пакета (как mattermost.go
// + plugin.go): сам этот файл — ядро (Handler, Register) и работа с единицей
// (список/карточка/дерево/состав/события), остальное — рядом:
// equipment_types.go — справочники (типы, магазины),
// equipment_templates.go — шаблоны комплектов,
// equipment_assignment.go — выдача/возврат/статус/дефект/списание,
// equipment_acts.go — акты и их файлы.

type Handler struct {
	service services.Equipment
	// maxSize — предел размера подписанного акта: multipart отбрасывается еще на
	// парсинге тела, иначе большой файл успел бы лечь в память/на диск.
	maxSize int64
}

func NewHandler(service services.Equipment, maxSize int64) *Handler {
	return &Handler{service: service, maxSize: maxSize}
}

func Register(api *gin.RouterGroup, service services.Equipment, maxSize int64, middleware *middleware.Middleware) {
	handlers := NewHandler(service, maxSize)

	equipment := api.Group("/equipment", middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Read()))
	{
		equipment.GET("", handlers.getAll)
		equipment.GET("/:id", handlers.getByID)
		equipment.GET("/:id/children", handlers.getChildren)
		equipment.GET("/:id/assignments", handlers.getAssignmentHistory)
		equipment.GET("/:id/events", handlers.getEvents)
		equipment.GET("/:id/composition", handlers.getComposition)
		equipment.GET("/:id/composition/history", handlers.getCompositionHistory)
		equipment.GET("/tree", handlers.getTree)
		equipment.GET("/stats", handlers.getStats)
		equipment.GET("/types", handlers.getTypes)
		equipment.GET("/types/:id", handlers.getTypeByID)
		equipment.GET("/shops", handlers.getShops)
		equipment.GET("/templates", handlers.getTemplates)
		equipment.GET("/templates/:id", handlers.getTemplateByID)
		equipment.GET("/acts", handlers.getActs)
		equipment.GET("/acts/:id", handlers.getActByID)
		equipment.GET("/acts/:id/document", handlers.getActDocument)
		equipment.GET("/acts/:id/signed", handlers.getActSignedFile)

		equipment.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Write()))
		equipment.POST("", handlers.create)
		equipment.PUT("/:id", handlers.update)
		equipment.POST("/:id/assign", handlers.assign)
		equipment.POST("/:id/unassign", handlers.unassign)
		equipment.POST("/:id/status", handlers.changeStatus)
		equipment.POST("/:id/defect", handlers.addDefect)
		equipment.POST("/:id/write-off", handlers.writeOff)
		equipment.POST("/:id/composition", handlers.addComponents)
		equipment.DELETE("/:id/composition", handlers.removeComponents)
		equipment.POST("/types", handlers.createType)
		equipment.PUT("/types/:id", handlers.updateType)
		equipment.POST("/templates", handlers.createTemplate)
		equipment.PUT("/templates/:id", handlers.updateTemplate)
		equipment.POST("/acts", handlers.createAct)
		equipment.POST("/acts/:id/signed", handlers.uploadActSigned)

		equipment.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceEquipment).Delete()))
		equipment.DELETE("/:id", handlers.delete)
		equipment.DELETE("/types/:id", handlers.deleteType)
		equipment.DELETE("/templates/:id", handlers.deleteTemplate)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.EquipmentFilter{}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}
	filter.RealmID = realmID

	data, total, err := h.service.Get(c, filter)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, total)
}

func (h *Handler) getByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetByID(c, &models.GetEquipmentByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getChildren(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetChildren(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getTree(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.GetEquipmentTreeDTO{}
	if err := c.ShouldBindQuery(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	data, total, err := h.service.GetTree(c, dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, total)
}

func (h *Handler) getStats(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetStats(c, realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.CreateEquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.RealmID = realmID
	dto.CreatedBy = &actor.ID
	dto.Actor = actor

	data, err := h.service.Create(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: data.ID, Message: "Оборудование создано"})
}

func (h *Handler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if dto.ID == nil || id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = &id
	dto.RealmID = realmID
	dto.Actor = actor

	if _, err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование обновлено"})
}

func (h *Handler) delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c, &models.DelEquipmentDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getEvents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.EquipmentEventsFilter{}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}

	data, err := h.service.GetEvents(c, realmID, id, filter.Limit, filter.Offset)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getComposition(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetComposition(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getCompositionHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetCompositionHistory(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) addComponents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.KitCompositionOpDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.AddComponents(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Состав комплекта обновлён"})
}

func (h *Handler) removeComponents(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.KitCompositionOpDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.ID = id
	dto.RealmID = realmID
	dto.Actor = actor

	if err := h.service.RemoveComponents(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Состав комплекта обновлён"})
}
