package categories

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

type modifierHandler struct {
	service services.CategoryModifiers
}

// registerModifiers — CRUD подкатегорий. Отдельный префикс, а не вложенный
// путь под /categories/:id, чтобы не конфликтовать с GET /categories/:id.
// Права переиспользуют ресурс category: write даётся начальнику области.
func registerModifiers(api *gin.RouterGroup, service services.CategoryModifiers, mw *middleware.Middleware) {
	h := &modifierHandler{service: service}
	group := api.Group("/category-modifiers", mw.CheckPermissions(access.Reg.R(access.ResourceCategory).Read()))
	{
		group.GET("", h.getAll)
		group.GET("/:id", h.getByID)

		group.Use(mw.CheckPermissions(access.Reg.R(access.ResourceCategory).Write()))
		group.POST("", h.create)
		group.PUT("/:id", h.update)

		group.Use(mw.CheckPermissions(access.Reg.R(access.ResourceCategory).Delete()))
		group.DELETE("/:id", h.delete)
	}
}

func (h *modifierHandler) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	req := &models.GetCategoryModifiersDTO{RealmID: realmID}
	if raw := c.Query("categoryId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
			return
		}
		req.CategoryID = &id
	}

	data, err := h.service.Get(c, req)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *modifierHandler) getByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	data, err := h.service.GetByID(c, &models.GetCategoryModifierByIDDTO{ID: id, RealmID: realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *modifierHandler) create(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.CategoryModifierDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.RealmID = realmID
	dto.ID = nil

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusCreated, response.IdResponse{Message: "Подкатегория создана"})
}

func (h *modifierHandler) update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.CategoryModifierDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	dto.ID = &id
	dto.RealmID = realmID

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Подкатегория обновлена"})
}

func (h *modifierHandler) delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c, &models.DelCategoryModifierDTO{ID: id, RealmID: realmID}); err != nil {
		response.SendError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
