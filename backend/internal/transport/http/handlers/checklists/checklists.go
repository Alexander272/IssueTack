package checklists

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

type Handler struct {
	service services.Checklists
}

func NewHandler(service services.Checklists) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Checklists, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	checklists := api.Group("/checklists", middleware.CheckPermissions(access.Reg.R(access.ResourceChecklist).Read()))
	{
		checklists.GET("", handlers.getAll)
		checklists.GET("/:id", handlers.getByID)
		checklists.GET("/:id/items", handlers.getItems)

		write := checklists.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceChecklist).Write()))
		{
			write.POST("", handlers.create)
			write.PUT("/:id", handlers.update)
			write.PUT("/:id/items", handlers.setItems)
			write.POST("/:id/apply/:ticketId", handlers.apply)
		}

		delete := checklists.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceChecklist).Delete()))
		{
			delete.DELETE("/:id", handlers.delete)
		}
	}
}

func (h *Handler) getAll(c *gin.Context) {
	dto := &models.GetChecklistTemplatesDTO{}

	if realmID := c.Query("realmId"); realmID != "" {
		id, err := uuid.Parse(realmID)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		dto.RealmID = id
	}

	data, err := h.service.Get(c, dto)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByID(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	data, err := h.service.GetByID(c, &models.GetChecklistTemplateDTO{ID: id})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) getItems(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	data, err := h.service.GetItems(c, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.ChecklistTemplateDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Id: dto.ID, Message: "Шаблон создан"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.ChecklistTemplateDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if id != dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = id

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Id: dto.ID, Message: "Шаблон обновлен"})
}

func (h *Handler) setItems(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	var items []*models.ChecklistTemplateItemDTO
	if err := c.BindJSON(&items); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.SetItems(c, nil, id, items); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Элементы шаблона обновлены"})
}

func (h *Handler) apply(c *gin.Context) {
	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	ticketID, err := uuid.Parse(c.Param("ticketId"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	if err := h.service.ApplyTemplate(c, nil, ticketID, templateID, actor); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Шаблон применен"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.service.Delete(c, &models.DelChecklistTemplateDTO{ID: id}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
