import { useCheckPermissions } from '../hooks/checkPerms'

export const useCan = (permission: string) =>
	useCheckPermissions({
		anyOf: [permission],
	})
