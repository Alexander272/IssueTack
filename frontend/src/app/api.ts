export const API = {
	auth: {
		signIn: `auth/sign-in` as const,
		refresh: `auth/refresh` as const,
		signOut: `auth/sign-out` as const,
	},

	realms: {
		base: 'realms',
		byId: (id: string) => `realms/${id}`,
	},

	users: {
		base: '/users' as const,
		sync: '/users/sync' as const,
		access: '/users/access' as const,
		logins: '/users/logins' as const,
	},
	roles: {
		base: '/roles' as const,
		stats: '/roles/all/stats' as const,
		permissions: (id: string) => `/roles/${id}/permissions` as const,
	},

	permissions: {
		base: '/permissions' as const,
		resources: '/permissions/resources' as const,
	},
	audit: '/audit' as const,
	statistics: {
		search: '/statistics/search' as const,
		priceSearch: '/prices/statistics/search/' as const,
		activity: '/statistics/activity' as const,
		logins: '/statistics/logins' as const,
	},
}
