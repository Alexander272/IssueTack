package roles

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
	service services.Roles
}

func NewHandler(service services.Roles) *Handler {
	return &Handler{
		service: service,
	}
}

func Register(api *gin.RouterGroup, service services.Roles, middleware *middleware.Middleware) {
	handlers := NewHandler(service)

	roles := api.Group("/roles", middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Read()))
	{
		roles.GET("", handlers.getAll)
		roles.GET("/all/stats", handlers.getWithStats)
		roles.GET("/:id", handlers.get)
		roles.GET("/:id/permissions", handlers.getWithPermissions)

		roles.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Write()))
		roles.POST("", handlers.create)
		roles.PUT("/:id", handlers.update)
		roles.PUT("/:id/permissions", handlers.setPermissions)

		roles.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceRole).Delete()))
		roles.DELETE("/:id", handlers.delete)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	roles, err := h.service.GetAll(c, &realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, roles, len(roles))
}

func (h *Handler) get(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	role, err := h.service.GetOne(c, &models.GetRoleDTO{ID: id, RealmID: &realmID})
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, role)
}

func (h *Handler) getWithStats(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	roles, err := h.service.GetWithStats(c, &realmID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, roles, len(roles))
}

func (h *Handler) getWithPermissions(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	role, err := h.service.GetOneWithPermissions(c, &models.GetRoleDTO{ID: id, RealmID: &realmID})
	if err != nil {
		response.SendError(c, err, id)
		return
	}
	response.SendData(c, role)
}

func (h *Handler) create(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	dto := &models.RoleDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	// realm только из контекста: тело запроса не может выбрать, в какую область
	// попадёт роль (иначе роль чужого реалма создавалась бы в «своём»).
	dto.RealmID = realmID

	// Новая роль всегда обычная: системность задаёт сервер, is_editable=true —
	// дефолт колонки (в INSERT не пишется).
	dto.IsSystem = false

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Create(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.JSON(http.StatusCreated, response.IdResponse{Message: "Роль создана"})
}

func (h *Handler) update(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.RoleDTO{}
	if err := c.BindJSON(dto); err != nil {
		response.SendError(c, err)
		return
	}
	if id != dto.ID {
		response.SendError(c, fmt.Errorf("%w: %s", models.ErrInvalidInput, "id is not equal to dto.ID"))
		return
	}
	dto.ID = id
	dto.RealmID = realmID

	// is_system/is_editable не трогаем: они не пишутся в UPDATE, иначе правка
	// системной роли снимала бы с неё защиту `AND NOT is_system` в удалении.
	// Системность роли меняется только сидом, неприкасаемость — только созданием
	// роли с is_editable=false (в API это root, см. RoleRepo.Create).

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Update(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.JSON(http.StatusOK, response.IdResponse{Message: "Роль обновлена"})
}

func (h *Handler) delete(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	strId := c.Param("id")
	id, err := uuid.Parse(strId)
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}
	dto := &models.DeleteRoleDTO{ID: id, RealmID: &realmID}

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.Delete(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) setPermissions(c *gin.Context) {
	realmID, ok := utils.RequireRealmUUID(c)
	if !ok {
		return
	}

	var req struct {
		PermissionIDs []string `json:"permissionIds"`
	}
	if err := c.BindJSON(&req); err != nil {
		response.SendError(c, err)
		return
	}

	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.SendError(c, fmt.Errorf("%w: %v", models.ErrInvalidInput, err))
		return
	}

	dto := &models.SetPermissionsDTO{RoleID: roleID, PermissionIDs: req.PermissionIDs, RealmID: &realmID}

	if err := h.service.SetPermissions(c, dto); err != nil {
		response.SendError(c, err, req)
		return
	}

	c.JSON(http.StatusOK, response.IdResponse{Message: "Права роли обновлены"})
}
