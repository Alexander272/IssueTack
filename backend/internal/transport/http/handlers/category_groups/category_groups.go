package category_groups

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

// Handler — обработчики разделов категорий. Права берём от access.ResourceCategory:
// раздел — это часть справочника категорий, отдельный Casbin-ресурс не заводим.
type Handler struct {
	service services.CategoryGroups
}

func NewHandler(service services.CategoryGroups) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.CategoryGroups, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	groups := api.Group("/category-groups", middleware.CheckPermissions(access.Reg.R(access.ResourceCategory).Read()))
	{
		groups.GET("", handlers.getAll)
		groups.GET("/:id", handlers.getByID)

		groups.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceCategory).Write()))
		groups.POST("", handlers.create)
		groups.PUT("/:id", handlers.update)

		groups.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceCategory).Delete()))
		groups.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.Get(c, &models.GetCategoryGroupsDTO{RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByID(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetByID(c, &models.GetCategoryGroupByIdDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.CategoryGroupDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Раздел создан"})
}

func (h *Handler) update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.CategoryGroupDTO{}
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

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Раздел обновлён"})
}

func (h *Handler) delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	realmID, ok := utils.GetRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c, &models.DelCategoryGroupDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return uuid.Nil, false
	}
	return id, true
}
