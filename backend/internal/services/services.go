package services

import (
	"context"
	"log"

	"github.com/Alexander272/IssueTrack/backend/internal/config"
	"github.com/Alexander272/IssueTrack/backend/internal/events"
	"github.com/Alexander272/IssueTrack/backend/internal/repository"
	"github.com/Alexander272/IssueTrack/backend/pkg/auth"
	"github.com/Alexander272/IssueTrack/backend/pkg/mattermost"
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
	CategoryGroups
	CategoryModifiers
	Sites
	Tickets
	Statistics
	Subtasks
	Attachments
	Checklists
	Comments
	Notifications
	Subscriptions
	TicketFavorites
	ActivityLog
	UserRealms
	Equipment
	// EquipmentImport — импорт единиц оборудования из файла. Отдельный сервис,
	// а не метод сервиса оборудования: импорту нужны справочники площадок и
	// пользователей, которых у оборудования нет.
	EquipmentImport         *EquipmentImportService
	EquipmentPeriodicTasks  *EquipmentPeriodicTaskService
	EquipmentOperationTasks *EquipmentOperationTaskService
	// Интеграции
	Calendar WorkCalendar
	Scheduler
	Mattermost
}

// Deps — готовые внешние зависимости (конфиг, репозитории, keycloak),
// которые сервисы не создают сами.
type Deps struct {
	Ctx      context.Context
	Conf     *config.Config
	Repo     *repository.Repository
	Keycloak *auth.KeycloakClient
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
		Tickets:   deps.Repo.Tickets,
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

	// Объект доступа к тикетам — единственный держатель Casbin-политик среди домена;
	// все проверки прав (включая realm-supervisor и «управление») идут через него.
	access := NewTicketAccessService(deps.Repo.Tickets, groups, policies)

	session := NewSessionService(deps.Keycloak, policies, userRealms, users, groups, access, cacheSvc)

	categoryVisibility := NewCategoryVisibility(access, groups)
	categoryModifiers := NewCategoryModifierService(deps.Repo.CategoryModifiers, deps.Repo.Categories)
	categories := NewCategoryService(deps.Repo.Categories, deps.Repo.Tickets, deps.Repo.CategoryModifiers, categoryVisibility)
	categoryGroups := NewCategoryGroupService(deps.Repo.CategoryGroups, categoryVisibility)
	sites := NewSiteService(deps.Repo.Sites, deps.Repo.SiteLocations)
	equipment := NewEquipmentService(deps.Repo.Equipment, transaction, &deps.Conf.FileServer)
	equipment.SetERPConfig(deps.Conf.Equipment)
	equipment.SetLocations(deps.Repo.SiteLocations)
	// Импорт оборудования: создаёт единицы через сервис оборудования, а
	// разрешает площадки, расположения и получателей по справочникам.
	equipmentImport := NewEquipmentImportService(equipment, deps.Repo.Sites, deps.Repo.Users)
	equipmentImport.SetLocations(deps.Repo.SiteLocations)
	logs := NewActivityLogService(deps.Repo.ActivityLog, transaction)
	subtasks := NewSubtaskService(deps.Repo.Subtasks, logs, users, access)
	statistics := NewStatisticsService(deps.Repo.Statistics, groups, access)
	subscriptionOps := NewTicketSubscriptionOpsService(deps.Repo.TicketSubscriptions)
	calendar := NewWorkCalendar(&WorkCalendarDeps{
		Repo:           deps.Repo.Calendar,
		Conf:           deps.Conf.Calendar,
		WorkingHoursTZ: deps.Conf.Tickets.WorkingHours.Timezone,
	})

	// Канал уведомлений в Mattermost: DM от бота реалма. Собирается до NotificationService,
	// чтобы уведомления всех событий тикетов доставлялись в Mattermost через него.
	mmMost := mattermost.NewMost(mattermost.MostConfig{
		ServerURL: deps.Conf.Mattermost.URL,
		BaseURL:   deps.Conf.Http.BaseURL,
	})
	notifChan := NewMattermostNotifier(deps.Repo.Mattermost, users, mmMost, policies, deps.Conf.Http.BaseURL)
	notifications := NewNotificationService(&NotificationDeps{
		Repo:          deps.Repo.Notifications,
		Subscriptions: subscriptionOps,
		UserRealms:    userRealms,
		Groups:        groups,
		TxManager:     transaction,
		Channels:      []Notifier{notifChan},
	})
	attachments := NewAttachmentService(deps.Repo.Attachments, &deps.Conf.FileServer, access, subtasks)
	checklists := NewChecklistService(deps.Repo.Checklists, subtasks, policies)

	// Тикеты собираются до своих потребителей (comments/subscriptions/favorites),
	// чтобы те зависели от сервиса — владельца агрегата тикетов, а не от репозитория.
	tickets := NewTicketService(&TicketDeps{
		Repo:            deps.Repo.Tickets,
		TxManager:       transaction,
		Logs:            logs,
		Subtasks:        subtasks,
		Attachments:     attachments,
		Notifications:   notifications,
		Groups:          groups,
		Categories:      categories,
		Modifiers:       categoryModifiers,
		Access:          access,
		Users:           users,
		Sites:           sites,
		DueDateRequests: deps.Repo.DueDateRequests,
		Favorites:       deps.Repo.TicketFavorites,
		Calendar:        calendar,
	})

	// Оборудование заводит заявку в 1C:ERP при выдаче/возврате, а заявку создаёт
	// TicketService — цикл equipment → tickets разрывается поздним связыванием.
	equipment.SetTicketCreator(tickets)
	// Списание оборудования разрешает только начальник области, а эта проверка
	// живёт в сервисе доступа — тем же поздним связыванием.
	equipment.SetAccessChecker(access)

	// Периодические задачи оборудования: создают заявки по расписанию.
	equipmentPeriodicTasks := NewEquipmentPeriodicTaskService(deps.Repo.Equipment, transaction)
	equipmentPeriodicTasks.SetTicketCreator(tickets)
	equipmentPeriodicTasks.SetCategories(categories)
	equipmentPeriodicTasks.SetUsers(users)
	equipment.SetPeriodicStarter(equipmentPeriodicTasks)

	// Правила автоматических задач после выдачи/возврата оборудования:
	// настраиваются в БД через UI, при отсутствии правил работает конфиг.
	equipmentOperationTasks := NewEquipmentOperationTaskService(deps.Repo.Equipment, transaction)
	equipmentOperationTasks.SetTicketCreator(tickets)
	equipment.SetOperationTasks(equipmentOperationTasks)

	// Цикл tickets → attachments разорван: уведомление о вложении рассылает сам
	// TicketService (UploadAttachment), поэтому attachments не зависят от тикетов
	// и позднее связывание не требуется.
	subscriptions := NewTicketSubscriptionService(deps.Repo.TicketSubscriptions, tickets, access)
	favorites := NewTicketFavoritesService(deps.Repo.TicketFavorites, tickets, access)

	comments := NewCommentService(deps.Repo.Comments, access, tickets, users, deps.Repo.Mattermost, mmMost, notifications, transaction, attachments)

	// --- Кластер интеграций ---------------------------------------------
	audit.StartListening(deps.Ctx, updatePolicyEvent)

	mattermostSvc := NewMattermostService(&MattermostDeps{
		Repo:        deps.Repo.Mattermost,
		Users:       users,
		UserRealms:  userRealms,
		Roles:       roles,
		Realms:      realms,
		Tickets:     tickets,
		Groups:      groups,
		Categories:  categories,
		Sites:       sites,
		Attachments: attachments,
		Comments:    comments,
		Access:      access,
		Policies:    policies,
		EventBus:    updatePolicyEvent,
		Most:        mmMost,
		BaseURL:     deps.Conf.Http.BaseURL,
		Calendar:    calendar,
	})

	// Планировщик создаётся после MattermostService: автосинк пользователей
	// работает через тот же фасад, что и ручная кнопка/команда.
	scheduler := NewSchedulerService(&SchedulerDeps{
		Tickets:                tickets,
		Notifications:          notifications,
		Favorites:              favorites,
		EquipmentReturns:       deps.Repo.Equipment,
		MMRepo:                 deps.Repo.Mattermost,
		Users:                  users,
		MMSender:               mmMost,
		EquipmentPeriodicTasks: equipmentPeriodicTasks,
		Calendar:               calendar,
		CalendarConf:           deps.Conf.Calendar,
		MattermostSync:         mattermostSvc,
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
		Groups:                  groups,
		Categories:              categories,
		CategoryGroups:          categoryGroups,
		CategoryModifiers:       categoryModifiers,
		Sites:                   sites,
		Tickets:                 tickets,
		Statistics:              statistics,
		Subtasks:                subtasks,
		Attachments:             attachments,
		Checklists:              checklists,
		Comments:                comments,
		Notifications:           notifications,
		Subscriptions:           subscriptions,
		TicketFavorites:         favorites,
		ActivityLog:             logs,
		UserRealms:              userRealms,
		Equipment:               equipment,
		EquipmentImport:         equipmentImport,
		EquipmentPeriodicTasks:  equipmentPeriodicTasks,
		EquipmentOperationTasks: equipmentOperationTasks,

		// Интеграции
		Calendar:   calendar,
		Scheduler:  scheduler,
		Mattermost: mattermostSvc,
	}
}
