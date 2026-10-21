package equipment

import (
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Шаблоны комплектов: сохранённый набор позиций, из которого собирают комплект.

func (h *Handler) getTemplates(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetKitTemplates(c, realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getTemplateByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetKitTemplateByID(c, &models.GetEquipmentKitTemplateByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) createTemplate(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentKitTemplateDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	if err := h.service.CreateKitTemplate(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Шаблон комплекта создан"})
}

func (h *Handler) updateTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentKitTemplateDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if dto.ID == nil || id != *dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = &id
	dto.RealmID = realmID

	if err := h.service.UpdateKitTemplate(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Шаблон комплекта обновлён"})
}

func (h *Handler) deleteTemplate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteKitTemplate(c, &models.DelEquipmentKitTemplateDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
