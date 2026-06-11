package access

const (
	ResourceRole     ResourceSlug = "role"
	ResourcePerm     ResourceSlug = "permission"
	ResourceAudit    ResourceSlug = "audit_log"
	ResourceRealm    ResourceSlug = "realm"
	ResourceGroup    ResourceSlug = "group"
	ResourceCategory ResourceSlug = "category"
	ResourceSite     ResourceSlug = "site"
	ResourceActivity ResourceSlug = "activity_log"
	ResourceTicket   ResourceSlug = "ticket"
)

var OrderOfResources = map[ResourceSlug]int{
	ResourceTicket:   1,
	ResourceCategory: 2,
	ResourceGroup:    3,
	ResourceSite:     4,
	ResourceActivity: 10,
	ResourceRealm:    20,
	ResourceRole:     21,
	ResourcePerm:     22,
	ResourceAudit:    22,
}

var Reg = NewRegistry(
	Resource{
		Slug:           ResourceRole,
		Name:           "Роли",
		Group:          "Администрирование",
		Description:    "Управление ролями пользователей",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourcePerm,
		Name:           "Права",
		Group:          "Администрирование",
		Description:    "Действия, которые доступны пользователю",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourceAudit,
		Name:           "Журнал изменений",
		Group:          "Логи",
		Description:    "История изменений прав доступа и разрешений",
		AllowedActions: actions(Read),
	},
	Resource{
		Slug:           ResourceRealm,
		Name:           "Области",
		Group:          "Администрирование",
		Description:    "Управление областями доступа (Realms)",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourceGroup,
		Name:           "Группы",
		Group:          "Администрирование",
		Description:    "Управление группами пользователей",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourceCategory,
		Name:           "Категории",
		Group:          "Администрирование",
		Description:    "Управление категориями заявок",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourceSite,
		Name:           "Площадки",
		Group:          "Администрирование",
		Description:    "Управление рабочими площадками",
		AllowedActions: actions(All),
	},
	Resource{
		Slug:           ResourceActivity,
		Name:           "Журнал активности",
		Group:          "Логи",
		Description:    "Системный журнал действий пользователей",
		AllowedActions: actions(Read),
	},
	Resource{
		Slug:           ResourceTicket,
		Name:           "Заявки",
		Group:          "Операции",
		Description:    "Работа с обращениями и заявками",
		AllowedActions: actions(All),
	},
)
