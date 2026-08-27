package services

import (
	"context"
	"log"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
	"github.com/Alexander272/IssueTrack/backend/pkg/ws_hub"
)

type Services struct {
	Realms
	Roles
	RoleHierarchy
	Permissions
	AuditLogs
	AccessPolicies
	Users
	Session

	Groups
	Categories
	Sites
	Tickets
	Subtasks
	Attachments
	Checklists
	Comments
	Notifications
	ActivityLog
	UserRealms
	Scheduler
	Mattermost
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

	perms, err := NewPermissionService(deps.Repo.Permissions, transaction, updatePolicyEvent)
	if err != nil {
		log.Fatalf("failed to initialize permission service: %s", err.Error())
	}
	rolesHierarchy := NewRoleHierarchyService(deps.Repo.RoleHierarchy)
	roles := NewRolesService(&RoleDeps{
		Repo:        deps.Repo.Roles,
		Realms:      deps.Repo.Realms,
		Hierarchy:   rolesHierarchy,
		Permissions: perms,
		EventBus:    updatePolicyEvent,
		TM:          transaction,
	})
	userRealms := NewUserRealmService(deps.Repo.UserRealms, transaction)
	users := NewUserService(&UsersDeps{
		Repo:      deps.Repo.Users,
		TxManager: transaction,
		UserRealm: userRealms,
		Keycloak:  deps.Keycloak,
		EventBus:  updatePolicyEvent,
	})

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

	groups := NewGroupService(deps.Repo.Groups, transaction)

	access := NewTicketAccessService(deps.Repo.Tickets, groups, policies)

	session := NewSessionService(deps.Keycloak, policies, userRealms, users, groups, cacheSvc)
	categories := NewCategoryService(deps.Repo.Categories)
	sites := NewSiteService(deps.Repo.Sites)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	subtasks := NewSubtaskService(deps.Repo.Subtasks, logs, access)
	attachments := NewAttachmentService(deps.Repo.Attachments, &deps.Conf.FileServer, access, deps.Repo.Subtasks)
	checklists := NewChecklistService(deps.Repo.Checklists, subtasks)
	notifications := NewNotificationService(deps.Hub, deps.Repo.Notifications, deps.Repo.Tickets, transaction)
	comments := NewCommentService(deps.Repo.Comments, access)
	tickets := NewTicketService(&TicketDeps{
		Repo:          deps.Repo.Tickets,
		TxManager:     transaction,
		Logs:          logs,
		Subtasks:      subtasks,
		Attachments:   attachments,
		Notifications: notifications,
		Groups:        groups,
		Policies:      policies,
		Access:        access,
	})

	audit.StartListening(deps.Ctx, updatePolicyEvent)
	scheduler := NewSchedulerService(&SchedulerDeps{Tickets: tickets})

	mmClient := mattermost.NewClient(deps.Conf.Mattermost.URL)
	mmMost := mattermost.NewMost(mmClient, mattermost.MostConfig{BaseURL: deps.Conf.Http.BaseURL})
	mattermostSvc := NewMattermostService(&MattermostDeps{
		Repo:        deps.Repo.Mattermost,
		Users:       users,
		UserRealms:  userRealms,
		Roles:       roles,
		Tickets:     tickets,
		Groups:      groups,
		Categories:  categories,
		Sites:       sites,
		Attachments: attachments,
		Client:      mmClient,
		Most:        mmMost,
		BaseURL:     deps.Conf.Http.BaseURL,
	})

	return &Services{
		Realms:         realms,
		AuditLogs:      audit,
		Roles:          roles,
		RoleHierarchy:  rolesHierarchy,
		Permissions:    perms,
		Users:          users,
		AccessPolicies: policies,
		Session:        session,

		Groups:        groups,
		Categories:    categories,
		Sites:         sites,
		Tickets:       tickets,
		Subtasks:      subtasks,
		Attachments:   attachments,
		Checklists:    checklists,
		Comments:      comments,
		Notifications: notifications,
		ActivityLog:   logs,
		UserRealms:    userRealms,
		Scheduler:     scheduler,
		Mattermost:    mattermostSvc,
	}
}
