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

// Справочники оборудования: типы (реквизит единицы) и магазины (подсказки для
// поля «Магазин»). Оба читаются списком на реалм, правятся по id.

func (h *Handler) getTypes(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetTypes(c, &models.GetEquipmentTypesDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getShops(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetShops(c, &models.GetEquipmentShopsDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getTypeByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetTypeByID(c, &models.GetEquipmentTypeByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) createType(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentTypeDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	if err := h.service.CreateType(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Тип оборудования создан"})
}

func (h *Handler) updateType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.EquipmentTypeDTO{}
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

	if err := h.service.UpdateType(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Тип оборудования обновлён"})
}

func (h *Handler) deleteType(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.DeleteType(c, &models.DelEquipmentTypeDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
