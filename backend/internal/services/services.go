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
	ActivityLog
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
	roles := NewRolesService(deps.Repo.Roles)
	rolesHierarchy := NewRoleHierarchyService(deps.Repo.RoleHierarchy)
	users := NewUserService(deps.Repo.Users, transaction)
	perms := NewPermissionService(deps.Repo.Permissions, transaction, updatePolicyEvent)
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
	tickets := NewTicketService(deps.Repo.Tickets, transaction, logs)

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
		ActivityLog:   logs,
	}
}
