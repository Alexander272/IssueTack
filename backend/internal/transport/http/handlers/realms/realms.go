package realms

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Alexander272/IssueTrack/backend/internal/access"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/models/response"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service   services.Realms
	mattermost services.Mattermost
}

func NewHandler(service services.Realms, mattermost services.Mattermost) *Handler {
	return &Handler{
		service:   service,
		mattermost: mattermost,
	}
}

func Register(api *gin.RouterGroup, svc services.Realms, mmSvc services.Mattermost, middleware *middleware.Middleware) {
	handlers := NewHandler(svc, mmSvc)

	realms := api.Group("/realms", middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Read()))
	{
		realms.GET("", handlers.getAll)
		realms.GET("/:id", handlers.getByID)

		realms.GET("/:id/mattermost", handlers.getMattermostSettings)

		realms.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Write()))
		realms.POST("", handlers.create)
		realms.PUT("/:id", handlers.update)

		realms.PUT("/:id/mattermost", handlers.saveMattermostSettings)
		realms.DELETE("/:id/mattermost", handlers.deleteMattermostSettings)

		realms.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Delete()))
		realms.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.GetAll(c)
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

	data, err := h.service.GetByID(c, &models.GetRealmByIdDTO{ID: id})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.RealmDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Область создана"})
}

func (h *Handler) update(c *gin.Context) {
	dto := &models.RealmDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
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
	c.JSON(http.StatusOK, response.IdResponse{Message: "Область обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.service.Delete(c, &models.DeleteRealmDTO{ID: id}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) getMattermostSettings(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	data, err := h.mattermost.GetSettings(c, id)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			response.SendData[*models.RealmMattermost](c, nil)
			return
		}
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) saveMattermostSettings(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.RealmMattermostDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.mattermost.SaveSettings(c, id, dto); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Настройки Mattermost сохранены"})
}

func (h *Handler) deleteMattermostSettings(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.mattermost.DeleteSettings(c, id); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
