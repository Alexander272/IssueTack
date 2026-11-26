package postgres

// const (
// 	SitesTable           = "public.sites"
// 	DomainsTable         = "public.domains"
// 	RolesTable           = "public.roles"
// 	RoleHierarchyTable   = "public.role_hierarchy"
// 	PermissionsTable     = "public.permissions"
// 	RolePermissionsTable = "public/role_permissions"
// 	UsersTable           = "public.users"
// 	UserRolesTable       = "public.user_roles"
// 	TicketsTable         = "public.tickets"
// )

var Tables = struct {
	Sites                          string
	Realms                         string
	Roles                          string
	RoleHierarchy                  string
	Permissions                    string
	RolePermissions                string
	Users                          string
	UserRoles                      string
	UserRealms                     string
	Groups                         string
	GroupMembers                   string
	Categories                     string
	CategoryGroups                 string
	CategoryModifiers              string
	Tickets                        string
	Subtasks                       string
	Attachments                    string
	ChecklistTemplates             string
	ChecklistTemplateItems         string
	Comments                       string
	ActivityLog                    string
	AuditLogs                      string
	Notifications                  string
	NotificationSettings           string
	TicketCounters                 string
	Mattermost                     string
	MattermostGreetings            string
	TicketSubscriptions            string
	TicketFavorites                string
	EquipmentTypes                 string
	EquipmentShops                 string
	Equipment                      string
	EquipmentAssignments           string
	EquipmentEvents                string
	EquipmentActs                  string
	KitCompositions                string
	EquipmentKitTemplates          string
	EquipmentKitTemplateItems      string
	EquipmentInventoryCounters     string
	EquipmentTypePeriodicTasks     string
	EquipmentPeriodicTaskInstances string
	EquipmentOperationTasks        string
	ReminderLastSent               string
	TicketDueDateRequests          string
	TicketNotifiers                string
	TicketDuplicateNotifiers       string
	ProductionCalendar             string
}{
	Sites:                          "sites",
	Realms:                         "realms",
	Roles:                          "roles",
	RoleHierarchy:                  "role_hierarchy",
	Permissions:                    "permissions",
	RolePermissions:                "role_permissions",
	Users:                          "users",
	UserRoles:                      "user_roles",
	UserRealms:                     "user_realms",
	Groups:                         "groups",
	GroupMembers:                   "group_members",
	Categories:                     "categories",
	CategoryGroups:                 "category_groups",
	CategoryModifiers:              "category_modifiers",
	Tickets:                        "tickets",
	Subtasks:                       "subtasks",
	Attachments:                    "attachments",
	ChecklistTemplates:             "checklist_templates",
	ChecklistTemplateItems:         "checklist_template_items",
	Comments:                       "comments",
	ActivityLog:                    "activity_logs",
	AuditLogs:                      "policy_audit_logs",
	Notifications:                  "notifications",
	NotificationSettings:           "user_notification_settings",
	TicketCounters:                 "ticket_counters",
	Mattermost:                     "realm_mattermost",
	MattermostGreetings:            "mattermost_greetings",
	TicketSubscriptions:            "ticket_subscriptions",
	TicketFavorites:                "ticket_favorites",
	EquipmentTypes:                 "equipment_types",
	EquipmentShops:                 "equipment_shops",
	Equipment:                      "equipment",
	EquipmentAssignments:           "equipment_assignments",
	EquipmentEvents:                "equipment_events",
	EquipmentActs:                  "equipment_acts",
	KitCompositions:                "kit_compositions",
	EquipmentKitTemplates:          "equipment_kit_templates",
	EquipmentKitTemplateItems:      "equipment_kit_template_items",
	EquipmentInventoryCounters:     "equipment_inventory_counters",
	EquipmentTypePeriodicTasks:     "equipment_type_periodic_tasks",
	EquipmentPeriodicTaskInstances: "equipment_periodic_task_instances",
	EquipmentOperationTasks:        "equipment_operation_tasks",
	ReminderLastSent:               "reminder_last_sent",
	TicketDueDateRequests:          "ticket_due_date_requests",
	TicketNotifiers:                "ticket_notifiers",
	TicketDuplicateNotifiers:       "ticket_duplicate_notifiers",
	ProductionCalendar:             "production_calendar",
}
