package users

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
	service services.Users
	session services.Session
}

func NewHandler(service services.Users, session services.Session) *Handler {
	return &Handler{
		service: service,
		session: session,
	}
}

func Register(api *gin.RouterGroup, services services.Users, session services.Session, middleware *middleware.Middleware) {
	handler := NewHandler(services, session)

	users := api.Group("/users", middleware.CheckPermissions(access.Reg.R(access.ResourceUser).Read()))
	{
		users.GET("/by-realm", handler.getByRealm)
		users.GET("/me/capabilities", handler.getCapabilities)
		users.GET("/:id", handler.getByID)

		users.Use(middleware.CheckPermissions(access.Reg.R(access.ResourceUser).Write()))
		users.GET("", handler.getAll)
		users.POST("/sync", handler.sync)
		users.PUT("/:id", handler.updateAccount)
	}
}

func (h *Handler) getAll(c *gin.Context) {
	data, err := h.service.GetAll(c, nil)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data, len(data))
}

func (h *Handler) getByRealm(c *gin.Context) {
	membership := models.MembershipFilter(c.Query("membership"))

	if membership == models.MembershipCustomers || membership == models.MembershipExecutors {
		realmID, ok := utils.GetRealmUUID(c)
		if !ok {
			return
		}
		data, err := h.service.GetByMembership(c, realmID, membership)
		if err != nil {
			response.SendError(c, err)
			return
		}
		response.SendData(c, data, len(data))
		return
	}

	var realmID *uuid.UUID
	if id, ok := utils.GetRealmUUID(c); ok {
		realmID = &id
	}

	data, err := h.service.GetAll(c, realmID)
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

	data, err := h.service.GetByID(c, id)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, data)
}

func (h *Handler) sync(c *gin.Context) {
	actor := utils.GetActor(c)
	if actor == nil {
		return
	}

	if err := h.service.Sync(c, actor); err != nil {
		response.SendError(c, err)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователи синхронизированы"})
}

func (h *Handler) updateAccount(c *gin.Context) {
	dto := &models.UpdateAccountDTO{}
	if err := utils.BindJSON(c, dto); err != nil {
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

	actor := utils.GetActor(c)
	if actor == nil {
		return
	}
	dto.Actor = actor

	if err := h.service.UpdateAccount(c, dto); err != nil {
		response.SendError(c, err, dto)
		return
	}
	c.JSON(http.StatusOK, response.IdResponse{Message: "Пользователь обновлен"})
}

func (h *Handler) getCapabilities(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		return
	}

	caps, err := h.session.GetAllCapabilities(c, user.ID)
	if err != nil {
		response.SendError(c, err)
		return
	}
	response.SendData(c, caps)
}
