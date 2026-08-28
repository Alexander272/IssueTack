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

// Services — композиционный корень всех бизнес-сервисов приложения.
// Поля сгруппированы по кластерам: доступ/аутентификация, домен, интеграции.
type Services struct {
	// Доступ и аутентификация
	Realms
	Roles
	RoleHierarchy
	Permissions
	AuditLogs
	AccessPolicies
	Users
	Session

	// Домен (каталог, группы, заявки)
	Groups
	Categories
	Sites
	Tickets
	Subtasks
	Attachments
	Checklists
	Comments
	Notifications
	Subscriptions
	ActivityLog
	UserRealms

	// Интеграции
	Scheduler
	Mattermost
}

// Deps — готовые внешние зависимости (конфиг, репозитории, keycloak, ws-хаб),
// которые сервисы не создают сами.
type Deps struct {
	Ctx      context.Context
	Conf     *config.Config
	Repo     *repository.Repository
	Keycloak *auth.KeycloakClient
	Hub      *ws_hub.Hub
}

// NewServices собирает все сервисы, разрешая их зависимости.
func NewServices(deps *Deps) *Services {
	transaction := NewTransactionManager(deps.Repo.Transaction)
	updatePolicyEvent := &events.PolicyEventManager{}

	// --- Кластер доступа и аутентификации -------------------------------
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

	// --- Кластер домена (группы, каталог, заявки) -----------------------
	// Группы создаются здесь же: они нужны и session (доступ), и тикетам.
	groups := NewGroupService(deps.Repo.Groups, deps.Repo.Tickets, transaction)
	session := NewSessionService(deps.Keycloak, policies, userRealms, users, groups, cacheSvc)

	access := NewTicketAccessService(deps.Repo.Tickets, groups, policies)

	categories := NewCategoryService(deps.Repo.Categories, deps.Repo.Tickets)
	sites := NewSiteService(deps.Repo.Sites)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	subtasks := NewSubtaskService(deps.Repo.Subtasks, logs, access)
	notifications := NewNotificationService(deps.Hub, deps.Repo.Notifications, deps.Repo.Tickets, deps.Repo.TicketSubscriptions, transaction)
	attachments := NewAttachmentService(deps.Repo.Attachments, &deps.Conf.FileServer, access, deps.Repo.Subtasks, notifications)
	checklists := NewChecklistService(deps.Repo.Checklists, subtasks)
	subscriptions := NewTicketSubscriptionService(deps.Repo.TicketSubscriptions, deps.Repo.Tickets, access)

	mmMost := mattermost.NewMost(mattermost.MostConfig{
		ServerURL: deps.Conf.Mattermost.URL,
		BaseURL:   deps.Conf.Http.BaseURL,
	})

	comments := NewCommentService(deps.Repo.Comments, access, deps.Repo.Tickets, users, deps.Repo.Mattermost, mmMost, notifications)
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

	// --- Кластер интеграций ---------------------------------------------
	audit.StartListening(deps.Ctx, updatePolicyEvent)
	scheduler := NewSchedulerService(&SchedulerDeps{Tickets: tickets})

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
		Comments:    comments,
		Most:        mmMost,
		BaseURL:     deps.Conf.Http.BaseURL,
	})

	return &Services{
		// Доступ и аутентификация
		Realms:         realms,
		AuditLogs:      audit,
		Roles:          roles,
		RoleHierarchy:  rolesHierarchy,
		Permissions:    perms,
		Users:          users,
		AccessPolicies: policies,
		Session:        session,

		// Домен
		Groups:        groups,
		Categories:    categories,
		Sites:         sites,
		Tickets:       tickets,
		Subtasks:      subtasks,
		Attachments:   attachments,
		Checklists:    checklists,
		Comments:      comments,
		Notifications: notifications,
		Subscriptions: subscriptions,
		ActivityLog:   logs,
		UserRealms:    userRealms,

		// Интеграции
		Scheduler:  scheduler,
		Mattermost: mattermostSvc,
	}
}
