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
	Realms: {
		Read: 'realm:read',
		Write: 'realm:write',
		Delete: 'realm:delete',
	},
	Roles: {
		Read: 'role:read',
		Write: 'role:write',
		Delete: 'role:delete',
	},
	Permissions: {
		Read: 'permission:read',
		Write: 'permission:write',
		Delete: 'permission:delete',
	},
	Tasks: {
		Read: 'ticket:read',
		Write: 'ticket:write',
		Delete: 'ticket:delete',
	},
	Equipment: {
		Read: 'equipment:read',
		Write: 'equipment:write',
		Delete: 'equipment:delete',
	},
})

// Права, дающие доступ к админскому разделу: достаточно любого из них.
export const AdminPermissions = [
	PermRules.Users.Write,
	PermRules.Roles.Write,
	PermRules.Realms.Write,
	PermRules.Permissions.Write,
]
