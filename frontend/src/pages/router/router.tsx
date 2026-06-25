import { createBrowserRouter, type RouteObject } from 'react-router'

import { AppRoutes } from './routes'
import { Layout } from '@/components/Layout/Layout'
import { NotFound } from '@/pages/notFound/NotFoundLazy'
import { Auth } from '@/pages/auth/AuthLazy'
import { Home } from '@/pages/home/HomeLazy'
import { Tasks } from '@/pages/tasks/TasksLazy'
import { Sites } from '@/pages/sites/SitesLazy'
import { Groups } from '@/pages/groups/GroupsLazy'
import { Categories } from '@/pages/categories/CategoriesLazy'
// import { Accesses } from '@/pages/accesses/AccessesLazy'
import { Dashboard } from '@/pages/accesses/dashboard/DashboardLazy'
import { Realms } from '@/pages/accesses/realms/RealmsLazy'
import { Users } from '@/pages/accesses/users/UsersLazy'
import { Role } from '@/pages/accesses/role/RoleLazy'
import { Permissions } from '@/pages/accesses/permissions/PermsLazy'
import PrivateRoute from './PrivateRoute'

const config: RouteObject[] = [
	{
		element: <Layout />,
		errorElement: <NotFound />,
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
						path: AppRoutes.Sites,
						element: <Sites />,
					},
					{
						path: AppRoutes.Groups,
						element: <Groups />,
					},
					{
						path: AppRoutes.Categories,
						element: <Categories />,
					},

					{
						path: AppRoutes.Accesses,
						children: [
							{
								index: true,
								element: <Dashboard />,
							},
							{
								path: AppRoutes.Realms,
								element: <Realms />,
							},
							{
								path: AppRoutes.UserAccess,
								element: <Users />,
							},
							{
								path: AppRoutes.RoleAccess,
								element: <Role />,
							},
							{
								path: AppRoutes.Permissions,
								element: <Permissions />,
							},
						],
					},
				],
			},
		],
	},
]

export const router = createBrowserRouter(config)
