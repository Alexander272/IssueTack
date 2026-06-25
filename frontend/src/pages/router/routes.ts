export const AppRoutes = Object.freeze({
	Home: '/' as const,
	Auth: '/auth' as const,

	Tasks: '/tasks' as const,
	Groups: '/groups' as const,
	Sites: '/sites' as const,
	Categories: '/categories' as const,

	Accesses: '/accesses' as const,
	Realms: '/accesses/realms' as const,
	UserAccess: '/accesses/user' as const,
	RoleAccess: '/accesses/roles' as const,
	Permissions: '/accesses/permissions' as const,
})
