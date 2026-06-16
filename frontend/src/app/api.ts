export const API = {
	auth: {
		signIn: `auth/sign-in` as const,
		refresh: `auth/refresh` as const,
		signOut: `auth/sign-out` as const,
	},
	search: {
		base: `search` as const,
		stream: `search/stream` as const,
	},
	orders: {
		base: `orders` as const,
		info: (id: string) => `orders/info/${id}` as const,
		byYear: (year: string) => `orders/by-year/${year}` as const,
		unique: (field: string) => `orders/unique/${field}` as const,
		flat: `orders/flat` as const,
		export: `/orders/export` as const,
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

	price: {
		base: '/prices' as const,
		search: '/prices/search' as const,
		searchAll: '/prices/search-all' as const,
		export: '/prices/export' as const,
		import: '/prices/import' as const,
		batch: '/prices/batch' as const,
	},
}
