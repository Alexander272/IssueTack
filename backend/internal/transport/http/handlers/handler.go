package handlers

import (
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/models"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/activity_log"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/attachments"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/audit_log"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/auth"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/categories"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/checklists"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/groups"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/notifications"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/permissions"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/realms"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/roles"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/sites"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/subtasks"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/tickets"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/users"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/middleware"
	"github.com/Alexander272/IssueTrack/backend/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	services   *services.Services
	conf       *config.Config
	middleware *middleware.Middleware
}

type Deps struct {
	Services   *services.Services
	Conf       *config.Config
	Middleware *middleware.Middleware
}

func NewHandler(deps *Deps) *Handler {
	return &Handler{
		services:   deps.Services,
		conf:       deps.Conf,
		middleware: deps.Middleware,
	}
}

func (h *Handler) Init(group *gin.RouterGroup) {
	v1 := group.Group("/v1")

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		err := v.RegisterValidation("enum", models.UniversalEnumValidator)
		if err != nil {
			logger.Error("register enum validator", logger.ErrAttr(err))
		}
	}

	auth.Register(v1, auth.Deps{Service: h.services.Session, Middleware: h.middleware, Auth: h.conf.Auth})
	secure := v1.Group("", h.middleware.VerifyToken)

	tickets.Register(secure, h.services.Tickets, h.middleware)
	subtasks.Register(secure, h.services.Subtasks, h.middleware)
	attachments.Register(secure, h.services.Attachments, h.middleware)
	checklists.Register(secure, h.services.Checklists, h.middleware)
	// comments.Register(secure, h.services.Comments, h.middleware)

	groups.Register(secure, h.services.Groups, h.middleware)
	categories.Register(secure, h.services.Categories, h.middleware)
	sites.Register(secure, h.services.Sites, h.middleware)

	permissions.Register(secure, h.services.Permissions, h.middleware)
	roles.Register(secure, h.services.Roles, h.middleware)
	realms.Register(secure, h.services.Realms, h.middleware)
	users.Register(secure, h.services.Users, h.middleware)

	notifications.Register(secure, h.services.Notifications)
	activity_log.Register(secure, h.services.ActivityLog, h.middleware)
	audit_log.Register(secure, h.services.AuditLogs, h.middleware)
}
