import { useAppSelector } from '@/hooks/redux'
import { getPermissionsSet } from '@/features/user/userSlice'
import { hasPermission } from '../utils/perms'

type CheckPermissionsParams = {
	anyOf?: string[]
	allOf?: string[]
}

export const useCheckPermissions = ({ anyOf, allOf }: CheckPermissionsParams): boolean => {
	const permissions = useAppSelector(getPermissionsSet)

	if (!permissions.size && (anyOf?.length || allOf?.length)) {
		return false
	}

	if (permissions.has('*') || permissions.has('*:*')) {
		return true
	}

	const anyAllowed = anyOf?.length ? anyOf.some(permission => hasPermission(permissions, permission)) : true

	const allAllowed = allOf?.length ? allOf.every(permission => hasPermission(permissions, permission)) : true

	return anyAllowed && allAllowed
	// if (anyOf?.length) {
	// 	return anyOf.some(permission => hasPermission(permissions, permission))
	// }

	// if (allOf?.length) {
	// 	return allOf.every(permission => hasPermission(permissions, permission))
	// }
	// return false
}
