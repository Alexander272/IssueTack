import { Outlet } from 'react-router'

import { useAppSelector } from '@/hooks/redux'
import { getCurrentCapabilities, getIsManager } from '@/features/user/userSlice'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { Forbidden } from '../forbidden/ForbiddenLazy'

interface RoleRouteProps {
	// Доступ только менеджерам (realm admin или менеджер групп)
	manager?: boolean
	// Доступ менеджерам или участникам групп
	managerOrMember?: boolean
	// Доступ по конкретному пермишену
	permission?: string
	// Доступ хотя бы по одному из пермишенов
	anyOfPermissions?: string[]
}

// Проверка ролевого/правового доступа к странице. Рендерит содержимое маршрута
// или экран <Forbidden /> при отсутствии доступа.
export default function RoleRoute({ manager, managerOrMember, permission, anyOfPermissions }: RoleRouteProps) {
	const isManager = useAppSelector(getIsManager)
	const { memberGroupIds } = useAppSelector(getCurrentCapabilities)
	const hasPermission = useCheckPermissions({
		anyOf: permission ? [permission] : anyOfPermissions,
	})

	if (managerOrMember && !isManager && memberGroupIds.length === 0) return <Forbidden />
	if (manager && !isManager) return <Forbidden />
	if (!manager && permission && !hasPermission) return <Forbidden />
	if (!manager && !permission && anyOfPermissions && !hasPermission) return <Forbidden />

	return <Outlet />
}
