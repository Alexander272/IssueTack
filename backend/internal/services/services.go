package services

import (
	"context"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
)

type Services struct {
	Realms
	Roles
	RoleHierarchy
	Permissions
	AuditLogs
	AccessPolices
	Users
	Session

	Groups
	Categories
	Sites
	Tickets
	Subtasks
	Attachments
	Checklists
	Notifications
	ActivityLog
	UserRealms
}

type Deps struct {
	Ctx      context.Context
	Conf     *config.Config
	Repo     *repository.Repository
	Keycloak *auth.KeycloakClient
	Hub      *ws_hub.Hub
}

func NewServices(deps *Deps) *Services {
	transaction := NewTransactionManager(deps.Repo.Transaction)

	updatePolicyEvent := &events.PolicyEventManager{}

	audit := NewAuditLogService(deps.Repo.AuditLogs, transaction)
	realms := NewRealmService(deps.Repo.Realms, transaction)

	perms := NewPermissionService(deps.Repo.Permissions, transaction, updatePolicyEvent)
	rolesHierarchy := NewRoleHierarchyService(deps.Repo.RoleHierarchy)
	roles := NewRolesService(&RoleDeps{
		Repo:        deps.Repo.Roles,
		Realms:      deps.Repo.Realms,
		Hierarchy:   rolesHierarchy,
		Permissions: perms,
		EventBus:    updatePolicyEvent,
		TM:          transaction,
	})
	users := NewUserService(&UsersDeps{
		Repo:      deps.Repo.Users,
		TxManager: transaction,
	})
	userRealms := NewUserRealmService(deps.Repo.UserRealms, transaction)

	cacheSvc := NewSessionCacheService(deps.Repo.SessionCache)
	adapter := NewAdapter(&AdapterDeps{
		Users:         users,
		RoleHierarchy: rolesHierarchy,
		Permissions:   perms,
		Ctx:           deps.Ctx,
	})
	policies := NewAccessPoliciesService(&PoliciesDeps{
		Conf:     deps.Conf.Casbin,
		Adapter:  adapter,
		EventBus: updatePolicyEvent,
		Cache:    cacheSvc,
	})

	session := NewSessionService(deps.Keycloak, policies, userRealms, users, cacheSvc)

	groups := NewGroupService(deps.Repo.Groups)
	categories := NewCategoryService(deps.Repo.Categories)
	sites := NewSiteService(deps.Repo.Sites)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	subtasks := NewSubtaskService(deps.Repo.Subtasks, logs)
	attachments := NewAttachmentService(deps.Repo.Attachments, &deps.Conf.FileServer)
	checklists := NewChecklistService(deps.Repo.Checklists, subtasks)
	notifications := NewNotificationService(deps.Hub, deps.Repo.Notifications, deps.Repo.Tickets, transaction)
	tickets := NewTicketService(deps.Repo.Tickets, transaction, logs, subtasks, attachments, notifications)

	audit.StartListening(updatePolicyEvent)

	return &Services{
		Realms:        realms,
		AuditLogs:     audit,
		Roles:         roles,
		RoleHierarchy: rolesHierarchy,
		Permissions:   perms,
		Users:         users,
		AccessPolices: policies,
		Session:       session,

		Groups:        groups,
		Categories:    categories,
		Sites:         sites,
		Tickets:       tickets,
		Subtasks:      subtasks,
		Attachments:   attachments,
		Checklists:    checklists,
		Notifications: notifications,
		ActivityLog:   logs,
		UserRealms:    userRealms,
	}
}
