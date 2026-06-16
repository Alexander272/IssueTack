import { Navigate, Outlet, useLocation } from 'react-router'

import { useAppSelector } from '@/hooks/redux'
import { getToken, getPermissions } from '@/features/user/userSlice'
import { Forbidden } from '../forbidden/ForbiddenLazy'
import { AppRoutes } from './routes'

// проверка авторизации пользователя
export default function PrivateRoute() {
	const token = useAppSelector(getToken)
	const permissions = useAppSelector(getPermissions)
	const location = useLocation()

	if (!token) return <Navigate to={AppRoutes.Auth} state={{ from: location }} />
	if (!permissions || !permissions.length) return <Forbidden />

	return <Outlet />
}
