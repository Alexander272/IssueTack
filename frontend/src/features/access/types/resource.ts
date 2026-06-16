export interface IResource {
	slug: ResourceSlug
	name: string
	group: string
	description: string
	/**
	 * Используем Partial<Record<...>>, чтобы не перечислять все экшены,
	 * если разрешены только некоторые.
	 */
	actions: Partial<Record<ActionCode, boolean>>
}
export type ResourceSlug = 'permission' | 'role' | 'user' | 'order' | 'audit_log' | 'activity_log'

export type ActionCode = 'read' | 'write' | 'delete' | '*'

// Константы для удобного использования в коде
export const Action = {
	Read: 'read' as const,
	Write: 'write' as const,
	Delete: 'delete' as const,
	All: '*' as const,
} as const

export const Resource = {
	Perm: 'permission' as ResourceSlug,
	Role: 'role' as ResourceSlug,
	User: 'user' as ResourceSlug,
	Order: 'order' as ResourceSlug,
	Audit: 'audit_log' as ResourceSlug,
	Activity: 'activity_log' as ResourceSlug,
} as const
