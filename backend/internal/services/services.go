package services

import (
	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
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
	conf *config.Config
	Repo *repository.Repository
	Hub  *ws_hub.Hub
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
	users := NewUserService(deps.Repo.Users, transaction)
	userRealms := NewUserRealmService(deps.Repo.UserRealms, transaction)

	adapter := NewAdapter(&AdapterDeps{
		Users:         users,
		RoleHierarchy: rolesHierarchy,
		Permissions:   perms,
	})
	policies := NewAccessPoliciesService(&PoliciesDeps{
		Conf:     deps.conf.Casbin,
		Adapter:  adapter,
		EventBus: updatePolicyEvent,
	})

	groups := NewGroupService(deps.Repo.Groups)
	categories := NewCategoryService(deps.Repo.Categories)
	sites := NewSiteService(deps.Repo.Sites)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	subtasks := NewSubtaskService(deps.Repo.Subtasks, logs)
	attachments := NewAttachmentService(deps.Repo.Attachments, &deps.conf.FileServer)
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
