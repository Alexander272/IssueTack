export const PermRules = Object.freeze({
	Orders: {
		Read: 'order:read',
		Write: 'order:write',
		Delete: 'order:delete',
	},
	Prices: {
		Read: 'price:read',
		Write: 'price:write',
	},
	Users: {
		Read: 'user:read',
		Write: 'user:write',
	},
	SearchLog: { Read: 'search_log:read' },
	PriceSearchLog: { Read: 'price_search_log:read' },
	ActivityLog: { Read: 'activity_log:read' },
	Logins: { Read: 'logins:read' },
	Permissions: {
		Read: 'permissions:read',
		Write: 'permissions:write',
		Delete: 'permissions:delete',
	},
	Tasks: {
		Write: 'ticket:write',
	},
})
