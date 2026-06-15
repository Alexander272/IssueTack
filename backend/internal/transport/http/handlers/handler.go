package handlers

import (
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/services"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/activity_log"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/attachments"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/audit_log"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/auth"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/categories"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/checklists"
	"github.com/Alexander272/IssueTrack/backend/internal/transport/http/handlers/comments"
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
	"github.com/gin-gonic/gin"
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

	auth.Register(v1, auth.Deps{Service: h.services.Session, Middleware: h.middleware, Auth: h.conf.Auth})
	secure := v1.Group("", h.middleware.VerifyToken)

	tickets.Register(secure, h.services.Tickets)
	subtasks.Register(secure, h.services.Subtasks)
	attachments.Register(secure, h.services.Attachments)
	checklists.Register(secure, h.services.Checklists)
	comments.Register(secure, h.services)

	groups.Register(secure, h.services.Groups, h.middleware)
	categories.Register(secure, h.services.Categories, h.middleware)
	sites.Register(secure, h.services.Sites, h.middleware)

	permissions.Register(secure, h.services.Permissions, h.middleware)
	roles.Register(secure, h.services.Roles, h.middleware)
	realms.Register(secure, h.services, h.middleware)
	users.Register(secure, h.services, h.middleware)

	notifications.Register(secure, h.services.Notifications)
	activity_log.Register(secure, h.services.ActivityLog, h.middleware)
	audit_log.Register(secure, h.services.AuditLogs, h.middleware)
}
