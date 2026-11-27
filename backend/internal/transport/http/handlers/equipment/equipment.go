package equipment

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Хендлеры оборудования разнесены по тематическим файлам пакета (как mattermost.go
// + plugin.go): регистрация маршрутов — в routes.go, ядро — здесь (единица,
// дерево, статистика, события), остальное рядом:
// equipment_composition.go — состав комплекта,
// equipment_types.go — справочники (типы, магазины),
// equipment_templates.go — шаблоны комплектов,
// equipment_assignment.go — выдача/возврат/статус/дефект/списание,
// equipment_acts.go — акты и их файлы,
// equipment_import.go — импорт из файла,
// equipment_periodic_tasks.go / equipment_operation_tasks.go — задачи.

type Handler struct {
	service services.Equipment
	// importer — импорт единиц оборудования из файла (xlsx/csv).
	importer EquipmentImporter
	// maxSize — предел размера подписанного акта: multipart отбрасывается еще на
	// парсинге тела, иначе большой файл успел бы лечь в память/на диск.
	maxSize int64
}

func NewHandler(deps Deps) *Handler {
	return &Handler{service: deps.Service, importer: deps.Importer, maxSize: deps.MaxSize}
}

// registerUnit — единица оборудования: список, карточка, дерево, статистика,
// события и CRUD. Операции со составом — в registerComposition.
func registerUnit(read, write, del *gin.RouterGroup, h *Handler) {
	read.GET("", h.getAll)
	read.GET("/:id", h.getByID)
	read.GET("/:id/children", h.getChildren)
	read.GET("/:id/events", h.getEvents)
	read.GET("/tree", h.getTree)
	read.GET("/stats", h.getStats)

	write.POST("", h.create)
	write.PUT("/:id", h.update)

	del.DELETE("/:id", h.delete)
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
