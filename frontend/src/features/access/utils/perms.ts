export const hasPermission = (permissions: Set<string>, required: string): boolean => {
	if (permissions.has('*') || permissions.has('*:*')) return true

	if (permissions.has(required)) return true

	const colonIndex = required.indexOf(':')
	if (colonIndex === -1) return false

	const resource = required.slice(0, colonIndex)
	const action = required.slice(colonIndex + 1)

	return (
		permissions.has(`${resource}:*`) || // orders:*
		permissions.has(`*:${action}`) // *:read
	)
}
