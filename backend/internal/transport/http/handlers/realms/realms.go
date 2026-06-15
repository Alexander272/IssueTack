package realms

import (
	"context"
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

type realmService interface {
	GetByID(ctx context.Context, req *models.GetRealmByIdDTO) (*models.Realm, error)
	Create(ctx context.Context, dto *models.RealmDTO) error
	Update(ctx context.Context, dto *models.RealmDTO) error
	Delete(ctx context.Context, dto *models.DeleteRealmDTO) error
}

type Handler struct {
	service realmService
}

func NewHandler(service realmService) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, svc *services.Services, middleware *middleware.Middleware) {
	rs, ok := svc.Realms.(*services.RealmService)
	if !ok {
		return
	}
	handlers := NewHandler(rs)

	realms := api.Group("/realms", middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Read()))
	{
		realms.GET("", handlers.getAll)
		realms.GET("/:id", handlers.getByID)

		write := realms.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Write()))
		{
			write.POST("", handlers.create)
			write.PUT("/:id", handlers.update)
		}

		delete := realms.Group("", middleware.CheckPermissions(access.Reg.R(access.ResourceRealm).Delete()))
		{
			delete.DELETE("/:id", handlers.delete)
		}
	}
}

func (h *Handler) getAll(c *gin.Context) {
	response.SendError(c, models.ErrNotFound)
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
