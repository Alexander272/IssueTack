export const API = {
	auth: {
		signIn: `auth/sign-in` as const,
		refresh: `auth/refresh` as const,
		signOut: `auth/sign-out` as const,
	},

	realms: {
		base: '/realms',
		byId: (id: string) => `/realms/${id}`,
		mattermost: (id: string) => `/realms/${id}/mattermost`,
	},

	users: {
		base: '/users' as const,
		sync: '/users/sync' as const,
		access: '/users/access' as const,
		logins: '/users/logins' as const,
		available: '/users/by-realm' as const,
		capabilities: '/users/me/capabilities' as const,
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
	activityLog: {
		base: '/activity-log' as const,
		byEntity: (entityId: string) => `/activity-log?entityId=${entityId}` as const,
	},
	statistics: {
		search: '/statistics/search' as const,
		priceSearch: '/prices/statistics/search/' as const,
		activity: '/statistics/activity' as const,
		logins: '/statistics/logins' as const,
	},

	categories: {
		base: '/categories' as const,
		byId: (id: string) => `/categories/${id}` as const,
	},
	groups: {
		base: '/groups' as const,
		byId: (id: string) => `/groups/${id}` as const,
	},
	tickets: {
		base: '/tickets' as const,
		byId: (id: string) => `/tickets/${id}` as const,
		subscription: (id: string) => `/tickets/${id}/subscription` as const,
	},
	favorites: {
		base: '/favorites' as const,
		byTicket: (id: string) => `/tickets/${id}/favorite` as const,
	},
	subtasks: {
		byTicket: (ticketId: string) => `/tickets/${ticketId}/subtasks` as const,
		byId: (ticketId: string, subtaskId: string) =>
			`/tickets/${ticketId}/subtasks/${subtaskId}` as const,
	},
	attachments: {
		upload: (entityType: string, entityId: string) =>
			`/attachments/${entityType}/${entityId}` as const,
		content: (id: string) => `/attachments/content/${id}` as const,
	},
	comments: {
		byTicket: (ticketId: string) => `/tickets/${ticketId}/comments` as const,
		delete: (ticketId: string, commentId: string) =>
			`/tickets/${ticketId}/comments/${commentId}` as const,
	},
	sites: {
		base: '/sites' as const,
		byId: (id: string) => `/sites/${id}` as const,
	},
	notifications: {
		settings: '/notifications/settings',
		getSettings: '/notifications',
	},
}
