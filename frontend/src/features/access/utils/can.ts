import { useCheckPermissions } from '../hooks/checkPerms'
import { useAppSelector } from '@/hooks/redux'
import { getPermissionsSet } from '@/features/user/userSlice'

export const useCan = (permission: string) =>
	useCheckPermissions({
		anyOf: [permission],
	})

export const useIsRoot = () => {
	const permissions = useAppSelector(getPermissionsSet)
	return permissions.has('*') || permissions.has('*:*')
}
