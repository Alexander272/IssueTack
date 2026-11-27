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

// EquipmentPeriodicTaskHandlers — HTTP-обработчики периодических задач оборудования.
type EquipmentPeriodicTaskHandlers struct {
	service services.EquipmentPeriodicTaskService
}

// NewEquipmentPeriodicTaskHandlers создаёт обработчики периодических задач.
func NewEquipmentPeriodicTaskHandlers(service services.EquipmentPeriodicTaskService) *EquipmentPeriodicTaskHandlers {
	return &EquipmentPeriodicTaskHandlers{service: service}
}

func registerPeriodicTasks(read, write *gin.RouterGroup, deps Deps) {
	if deps.PeriodicTasks == nil {
		return
	}

	h := NewEquipmentPeriodicTaskHandlers(*deps.PeriodicTasks)

	// Чтение — под equipment:read, изменение — под equipment:write.
	read.GET("/periodic-tasks", h.getTemplates)
	read.GET("/periodic-tasks/templates/:id", h.getTemplateByID)
	read.GET("/periodic-tasks/instances", h.getInstances)

	write.POST("/periodic-tasks/templates", h.createTemplate)
	write.PUT("/periodic-tasks/templates/:id", h.updateTemplate)
	write.DELETE("/periodic-tasks/templates/:id", h.deleteTemplate)
	write.POST("/periodic-tasks/run", h.runInstance)
}

// getTemplates — список шаблонов периодических задач.
// equipmentTypeId — необязательный фильтр по типу оборудования (uuid).
func (h *EquipmentPeriodicTaskHandlers) getTemplates(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	var equipmentTypeID *uuid.UUID
	if idStr := c.Query("equipmentTypeId"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.SendError(c, models.ErrNotValid)
			return
		}
		equipmentTypeID = &id
	}

	templates, err := h.service.GetPeriodicTaskTemplates(c.Request.Context(), realmID, equipmentTypeID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, templates)
}

// getTemplateByID — получение шаблона по ID.
func (h *EquipmentPeriodicTaskHandlers) getTemplateByID(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	template, err := h.service.GetPeriodicTaskTemplateByID(c.Request.Context(), id, realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, template)
}

// createTemplate — создание шаблона периодической задачи.
func (h *EquipmentPeriodicTaskHandlers) createTemplate(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	var dto models.EquipmentTypePeriodicTaskDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID
	// Автор шаблона — создатель заявок: задачи порождаются от него.
	if actor := utils.GetActor(c); actor != nil {
		dto.CreatedBy = &actor.ID
	}

	template, err := h.service.CreatePeriodicTaskTemplate(c.Request.Context(), &dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, template)
}

// updateTemplate — обновление шаблона периодической задачи.
func (h *EquipmentPeriodicTaskHandlers) updateTemplate(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	var dto models.EquipmentTypePeriodicTaskDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = &id
	dto.RealmID = realmID

	template, err := h.service.UpdatePeriodicTaskTemplate(c.Request.Context(), &dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, template)
}

// deleteTemplate — удаление шаблона периодической задачи.
func (h *EquipmentPeriodicTaskHandlers) deleteTemplate(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.SendError(c, models.ErrNotValid)
		return
	}

	err = h.service.DeletePeriodicTaskTemplate(c.Request.Context(), &models.DelEquipmentTypePeriodicTaskDTO{
		ID:      id,
		RealmID: realmID,
	})
	if err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// getInstances — список экземпляров периодических задач.
func (h *EquipmentPeriodicTaskHandlers) getInstances(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	filter := &models.GetEquipmentPeriodicTaskInstancesFilter{RealmID: realmID}
	if err := c.ShouldBindQuery(filter); err != nil {
		response.SendError(c, err)
		return
	}

	instances, total, err := h.service.GetPeriodicTaskInstances(c.Request.Context(), filter)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, instances, total)
}

// runInstance — ручной запуск периодической задачи.
func (h *EquipmentPeriodicTaskHandlers) runInstance(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	var dto models.ManualPeriodicTaskRunDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	actor := utils.GetActor(c)

	err := h.service.ManualRunTask(c.Request.Context(), &dto, actor)
	if err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
