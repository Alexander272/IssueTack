import type { ReactNode } from 'react'

import { AppRoutes } from '@/pages/router/routes'
import { DashboardIcon } from '../Icons/DashboardIcon'
import { UserIcon } from '../Icons/UserIcon'
import { ShieldLockIcon } from '../Icons/ShieldLockIcon'
import { AccessHandleIcon } from '../Icons/AccessHandleIcon'
import { LocalNetworkIcon } from '../Icons/LocalNetworkIcon'
import { DoubleRightIcon } from '../Icons/DoubleRightIcon'
import { InboxIcon } from '../Icons/InboxIcon'

export interface SidebarItem {
	path: string
	label: string
	icon: ReactNode
}

export interface SidebarConfig {
	items: SidebarItem[]
}

export interface SidebarRule {
	match: (path: string) => boolean
	config: SidebarConfig
}

const homeItems: SidebarItem[] = [{ path: AppRoutes.Home, label: 'Задачи', icon: <InboxIcon sx={{ fontSize: 18 }} /> }]

const accessesItems: SidebarItem[] = [
	{
		path: AppRoutes.Home,
		label: 'Главная',
		icon: <DoubleRightIcon sx={{ fill: '#000', fontSize: 18, transform: 'rotate(180deg)' }} />,
	},
	{ path: AppRoutes.Accesses, label: 'Дашборд', icon: <DashboardIcon sx={{ fontSize: 18 }} /> },
	{
		path: AppRoutes.Realms,
		label: 'Области',
		icon: <LocalNetworkIcon sx={{ fill: '#000', fontSize: 22 }} />,
	},
	{
		path: AppRoutes.UserAccess,
		label: 'Пользователи',
		icon: <UserIcon sx={{ fill: '#000', fontSize: 22 }} />,
	},
	{ path: AppRoutes.RoleAccess, label: 'Роли', icon: <ShieldLockIcon sx={{ fontSize: 22 }} /> },
	{ path: AppRoutes.Permissions, label: 'Права доступа', icon: <AccessHandleIcon fontSize='small' /> },
]

export const sidebarRules: SidebarRule[] = [
	{
		match: path => [AppRoutes.Home, '/tasks', '/categories', '/history', '/favorites'].includes(path),
		config: { items: homeItems },
	},
	{
		match: path => path.startsWith(AppRoutes.Accesses),
		config: { items: accessesItems },
	},
]
