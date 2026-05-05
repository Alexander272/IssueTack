package constants

type Resource struct {
	Code  string
	Name  string
	Group string
	Desc  string
}

const (
	ResRole     = "role"
	ResPerm     = "permission"
	ResAudit    = "audit_log"
	ResRealm    = "realm"
	ResGroup    = "group"
	ResCategory = "category"
	ResSite     = "site"
	ResActivity = "activity_log"
	ResTicket   = "ticket"
)

type ResRegistry struct {
	// Поля для быстрого доступа в коде (Check Permission)
	Role        Resource
	Permissions Resource
	AuditLog    Resource
	Realms      Resource
	Groups      Resource
	Categories  Resource
	Sites       Resource
	ActivityLog Resource
	Tickets     Resource

	// Слайс для генерации UI (Menu/Selects)
	Items []Resource
}

func NewResourceList() ResRegistry {
	// Сначала определяем отдельные объекты
	role := Resource{ResRole, "Роли", "system", ""}
	perms := Resource{ResPerm, "Разрешения", "system", "Действия которые доступны пользователю"}
	audit := Resource{ResAudit, "Журнал изменений", "logs", "Журнал в котором отслеживаются изменения разрешений"}
	realm := Resource{ResRealm, "Область", "system", ""}
	group := Resource{ResGroup, "Группа", "service", "Группа пользователей которым доступны заявки"}
	category := Resource{ResCategory, "Категория", "service", "Категория заявок"}
	site := Resource{ResSite, "Площадка", "service", ""}
	activity := Resource{ResActivity, "Журнал изменений", "logs", "Журнал в котором отслеживаются изменения заявок"}
	ticket := Resource{ResTicket, "Заявка", "service", ""}

	return ResRegistry{
		Role:        role,
		Permissions: perms,
		AuditLog:    audit,
		Realms:      realm,
		Groups:      group,
		Categories:  category,
		Sites:       site,
		ActivityLog: activity,
		Tickets:     ticket,

		// Собираем их в список один раз при инициализации
		Items: []Resource{
			role,
			perms,
			audit,
			realm,
			group,
			category,
			site,
			activity,
			ticket,
		},
	}
}

var Resources = NewResourceList()
