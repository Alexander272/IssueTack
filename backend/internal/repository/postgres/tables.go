package postgres

// const (
// 	SitesTable           = "public.sites"
// 	DomainsTable         = "public.domains"
// 	RolesTable           = "public.roles"
// 	RoleHierarchyTable   = "public.role_hierarchy"
// 	PermissionsTable     = "public.permissions"
// 	RolePermissionsTable = "public.role_permissions"
// 	UsersTable           = "public.users"
// 	UserRolesTable       = "public.user_roles"
// 	TicketsTable         = "public.tickets"
// )

var Tables = struct {
	Sites                  string
	Realms                 string
	Roles                  string
	RoleHierarchy          string
	Permissions            string
	RolePermissions        string
	Users                  string
	UserRoles              string
	UserRealms             string
	Groups                 string
	GroupMembers           string
	Categories             string
	Tickets                string
	Subtasks               string
	Attachments            string
	ChecklistTemplates     string
	ChecklistTemplateItems string
	Comments               string
	ActivityLog            string
	AuditLogs              string
	Notifications          string
	NotificationSettings   string
	TicketCounters         string
}{
	Sites:                  "sites",
	Realms:                 "realms",
	Roles:                  "roles",
	RoleHierarchy:          "role_hierarchy",
	Permissions:            "permissions",
	RolePermissions:        "role_permissions",
	Users:                  "users",
	UserRoles:              "user_roles",
	UserRealms:             "user_realms",
	Groups:                 "groups",
	GroupMembers:           "group_members",
	Categories:             "categories",
	Tickets:                "tickets",
	Subtasks:               "subtasks",
	Attachments:            "attachments",
	ChecklistTemplates:     "checklist_templates",
	ChecklistTemplateItems: "checklist_template_items",
	Comments:               "comments",
	ActivityLog:            "activity_log",
	AuditLogs:              "policy_audit_logs",
	Notifications:          "notifications",
	NotificationSettings:   "user_notification_settings",
	TicketCounters:         "ticket_counters",
}
