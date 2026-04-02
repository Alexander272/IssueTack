package constants

type Resources struct {
	Code string
	Name string
	Desc string
}

type ResList struct {
	Role        Resources
	Permissions Resources
	AuditLog    Resources
	Realms      Resources
	Groups      Resources
	Categories  Resources
	Sites       Resources
	ActivityLog Resources
	Tickets     Resources
}

var ResourcesList = ResList{
	Role:        Resources{"role", "Роли", ""},
	Permissions: Resources{"permission", "Разрешения", "Действия которые доступны пользователю"},
	AuditLog:    Resources{"audit_log", "Журнал изменений", "Журнал в котором отслеживаются изменения разрешений"},
	Realms:      Resources{"realm", "Область", ""},
	Groups:      Resources{"group", "Группа", "Группа пользователей которым доступны заявки"},
	Categories:  Resources{"category", "Категория", "Категория заявок"},
	Sites:       Resources{"site", "Площадка", ""},
	ActivityLog: Resources{"activity_log", "Журнал изменений", "Журнал в котором отслеживаются изменения заявок"},
	Tickets:     Resources{"ticket", "Заявка", ""},
}
