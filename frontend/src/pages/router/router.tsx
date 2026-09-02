import { createBrowserRouter, type RouteObject } from 'react-router'

import { AppRoutes } from './routes'
import { Layout } from '@/components/Layout/Layout'
import ErrorPage from '@/pages/error/ErrorPage'
import { Auth } from '@/pages/auth/AuthLazy'
import { Home } from '@/pages/home/HomeLazy'
import { Tasks } from '@/pages/tasks/TasksLazy'
import { TaskDetail } from '@/pages/tasks/TaskDetailLazy'
import { Favorites } from '@/pages/favorites/FavoritesLazy'
import { Sites } from '@/pages/sites/SitesLazy'
import { Groups } from '@/pages/groups/GroupsLazy'
import { Categories } from '@/pages/categories/CategoriesLazy'
import { NotificationSettings } from '@/pages/settings/NotificationSettingsLazy'
// import { Accesses } from '@/pages/accesses/AccessesLazy'
import { Dashboard } from '@/pages/accesses/dashboard/DashboardLazy'
import { Realms } from '@/pages/accesses/realms/RealmsLazy'
import { Users } from '@/pages/accesses/users/UsersLazy'
import { Role } from '@/pages/accesses/role/RoleLazy'
import { Permissions } from '@/pages/accesses/permissions/PermsLazy'
import PrivateRoute from './PrivateRoute'
import RoleRoute from './RoleRoute'

const config: RouteObject[] = [
	{
		element: <Layout />,
		errorElement: <ErrorPage />,
		children: [
			{
				path: AppRoutes.Auth,
				element: <Auth />,
			},
			{
				path: AppRoutes.Home,
				element: <PrivateRoute />,
				children: [
					{
						index: true,
						element: <Home />,
					},
					{
						path: AppRoutes.Tasks,
						element: <Tasks />,
					},
					{
						path: AppRoutes.TaskDetail,
						element: <TaskDetail />,
					},
					{
						path: AppRoutes.Favorites,
						element: <Favorites />,
					},
					{
						path: AppRoutes.Sites,
						element: <RoleRoute manager />,
						children: [
							{
								index: true,
								element: <Sites />,
							},
						],
					},
					{
						path: AppRoutes.Groups,
						element: <RoleRoute manager />,
						children: [
							{
								index: true,
								element: <Groups />,
							},
						],
					},
					{
						path: AppRoutes.Categories,
						element: <RoleRoute manager />,
						children: [
							{
								index: true,
								element: <Categories />,
							},
						],
					},
					{
						path: AppRoutes.NotificationSettings,
						element: <RoleRoute manager />,
						children: [
							{
								index: true,
								element: <NotificationSettings />,
							},
						],
					},

					{
						path: AppRoutes.Accesses,
						children: [
							{
								element: (
									<RoleRoute
										anyOfPermissions={['user:write', 'role:write', 'realm:write', 'permission:write']}
									/>
								),
								children: [
									{
										index: true,
										element: <Dashboard />,
									},
									{
										path: AppRoutes.Realms,
										element: <RoleRoute permission='realm:write' />,
										children: [{ index: true, element: <Realms /> }],
									},
									{
										path: AppRoutes.UserAccess,
										element: <RoleRoute permission='user:write' />,
										children: [{ index: true, element: <Users /> }],
									},
									{
										path: AppRoutes.RoleAccess,
										element: <RoleRoute permission='role:write' />,
										children: [{ index: true, element: <Role /> }],
									},
									{
										path: AppRoutes.Permissions,
										element: <RoleRoute permission='permission:write' />,
										children: [{ index: true, element: <Permissions /> }],
									},
								],
							},
						],
					},
				],
			},
		],
	},
]

export const router = createBrowserRouter(config)
