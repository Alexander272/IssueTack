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

// Жизненный цикл единицы: выдача, возврат, смена статуса, дефект и списание, а
// также история выдач. Все операции меняют состояние, поэтому требуют actor'а.

func (h *Handler) assign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.AssignEquipmentDTO{}
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

	if err := h.service.Assign(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование выдано"})
}

func (h *Handler) unassign(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.UnassignEquipmentDTO{}
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

	if err := h.service.Unassign(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: id, Message: "Оборудование возвращено"})
}

func (h *Handler) changeStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.ChangeStatusDTO{}
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

	data, err := h.service.ChangeStatus(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) addDefect(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.AddDefectDTO{}
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

	data, err := h.service.AddDefect(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) writeOff(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.WriteOffEquipmentDTO{}
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

	data, err := h.service.WriteOff(c, dto)
	if err != nil {
		response.SendError(c, err, dto)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getAssignmentHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetAssignmentHistory(c, realmID, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}
