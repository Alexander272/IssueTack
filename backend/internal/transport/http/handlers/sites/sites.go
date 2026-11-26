package sites

import (
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
	service services.Sites
}

func NewHandler(service services.Sites) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Sites, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	sites := api.Group("/sites", middleware.CheckPermissions(access.Reg.R(access.ResourceSite).Read()))
	{
		sites.GET("", handlers.getAll)
		// Статический путь должен быть раньше /:id, иначе Gin разберёт
		// all-locations как идентификатор площадки.
		sites.GET("/all-locations", handlers.getAllLocations)
		sites.GET("/:id", handlers.getByID)
		// Расположения читаются теми же правами, что и сама площадка: это её
		// реквизит, а не отдельный ресурс.
		sites.GET("/:id/locations", handlers.getLocations)

		sites.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceSite).Write()))
		sites.POST("", handlers.create)
		sites.PUT("/:id", handlers.update)
		sites.POST("/:id/locations", handlers.createLocation)
		sites.PUT("/:id/locations/:locationId", handlers.updateLocation)

		sites.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceSite).Delete()))
		sites.DELETE("/:id", handlers.delete)
		sites.DELETE("/:id/locations/:locationId", handlers.deleteLocation)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.Get(c, &models.GetSitesDTO{})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getAllLocations(c *gin.Context) {
	data, err := h.service.GetAllLocations(c)
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

	data, err := h.service.GetByID(c, &models.GetSiteByIdDTO{ID: id})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) create(c *gin.Context) {
	dto := &models.SiteDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Площадка создана"})
}

func (h *Handler) update(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.SiteDTO{}
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
	c.JSON(http.StatusOK, response.IdResponse{Message: "Площадка обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	if err := h.service.Delete(c, &models.DelSiteDTO{ID: id}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ===== Расположения площадки =====

func (h *Handler) getLocations(c *gin.Context) {
	siteID, err := parseSiteID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	data, err := h.service.GetLocations(c, siteID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) createLocation(c *gin.Context) {
	siteID, err := parseSiteID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	dto := &models.SiteLocationDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.SiteID = siteID
	if err := h.service.CreateLocation(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Расположение добавлено"})
}

func (h *Handler) updateLocation(c *gin.Context) {
	siteID, err := parseSiteID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	locationID, err := parseLocationID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	dto := &models.SiteLocationDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID, dto.SiteID = locationID, siteID
	if err := h.service.UpdateLocation(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Расположение обновлено"})
}

func (h *Handler) deleteLocation(c *gin.Context) {
	siteID, err := parseSiteID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	locationID, err := parseLocationID(c)
	if err != nil {
		response.SendError(c, err)
		return
	}
	if err := h.service.DeleteLocation(c, &models.DelSiteLocationDTO{ID: locationID, SiteID: siteID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseSiteID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", models.ErrInvalidInput, err)
	}
	return id, nil
}

func parseLocationID(c *gin.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("locationId"))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", models.ErrInvalidInput, err)
	}
	return id, nil
}
