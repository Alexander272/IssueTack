package equipment

import (
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// EquipmentOperationTaskHandlers — HTTP-обработчики правил автоматических задач
// после выдачи/возврата оборудования.
type EquipmentOperationTaskHandlers struct {
	service *services.EquipmentOperationTaskService
}

func NewEquipmentOperationTaskHandlers(service *services.EquipmentOperationTaskService) *EquipmentOperationTaskHandlers {
	return &EquipmentOperationTaskHandlers{service: service}
}

func registerOperationTasks(read, write *gin.RouterGroup, deps Deps) {
	if deps.OperationTasks == nil {
		return
	}

	h := NewEquipmentOperationTaskHandlers(deps.OperationTasks)

	// Чтение — под equipment:read, изменение — под equipment:write.
	read.GET("/operation-tasks", h.getAll)
	read.GET("/operation-tasks/:id", h.getByID)

	write.POST("/operation-tasks", h.create)
	write.PUT("/operation-tasks/:id", h.update)
	write.DELETE("/operation-tasks/:id", h.delete)
}

// getAll — список правил реалма; ?operation=assign|unassign сужает выборку.
func (h *EquipmentOperationTaskHandlers) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	var operation *models.EquipmentOperation
	if raw := c.Query("operation"); raw != "" {
		op := models.EquipmentOperation(raw)
		if !op.IsValid() {
			response.SendError(c, models.ErrNotValid)
			return
		}
		operation = &op
	}

	tasks, err := h.service.GetOperationTasks(c.Request.Context(), realmID, operation)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, tasks)
}

// getByID — правило по id.
func (h *EquipmentOperationTaskHandlers) getByID(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	task, err := h.service.GetOperationTaskByID(c.Request.Context(), id, realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, task)
}

// create — новое правило.
func (h *EquipmentOperationTaskHandlers) create(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentOperationTaskDTO{}
	if err := c.ShouldBindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	task, err := h.service.CreateOperationTask(c.Request.Context(), dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, task)
}

// update — правка правила.
func (h *EquipmentOperationTaskHandlers) update(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	dto := &models.EquipmentOperationTaskDTO{}
	if err := c.ShouldBindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = &id
	dto.RealmID = realmID

	if err := h.service.UpdateOperationTask(c.Request.Context(), dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// delete — удаление правила.
func (h *EquipmentOperationTaskHandlers) delete(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	if err := h.service.DeleteOperationTask(c.Request.Context(), &models.DelEquipmentOperationTaskDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
