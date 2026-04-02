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
	Sites           string
	Realms          string
	Roles           string
	RoleHierarchy   string
	Permissions     string
	RolePermissions string
	Users           string
	UserRoles       string
	Groups          string
	GroupMembers    string
	Categories      string
	Tickets         string
	Comments        string
	ActivityLog     string
}{
	Sites:           "public.sites",
	Realms:          "public.realms",
	Roles:           "public.roles",
	RoleHierarchy:   "public.role_hierarchy",
	Permissions:     "public.permissions",
	RolePermissions: "public.role_permissions",
	Users:           "public.users",
	UserRoles:       "public.user_roles",
	Groups:          "public.groups",
	GroupMembers:    "public.group_members",
	Categories:      "public.categories",
	Tickets:         "public.tickets",
	Comments:        "public.comments",
	ActivityLog:     "public.activity_log",
}
