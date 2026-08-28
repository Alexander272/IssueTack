import type { ReactNode } from 'react'

import { AppRoutes } from '@/pages/router/routes'
import {
	LayoutDashboardIcon,
	UserIcon,
	ShieldCheckIcon,
	KeyRoundIcon,
	NetworkIcon,
	ChevronsRightIcon,
	InboxIcon,
	LayersIcon,
	UsersIcon,
	BuildingIcon,
	SendIcon,
	SettingsIcon,
} from 'lucide-mui'

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

const homeItems: SidebarItem[] = [
	{ path: AppRoutes.Home, label: 'Заявки', icon: <SendIcon sx={{ fontSize: 20 }} /> },
	{ path: AppRoutes.Tasks, label: 'Задачи', icon: <InboxIcon sx={{ fontSize: 18 }} /> },
	{ path: AppRoutes.Groups, label: 'Группы', icon: <UsersIcon sx={{ fontSize: 18 }} /> },
	{ path: AppRoutes.Categories, label: 'Категории', icon: <LayersIcon sx={{ fontSize: 18 }} /> },
	{ path: AppRoutes.Sites, label: 'Площадки', icon: <BuildingIcon sx={{ fontSize: 18 }} /> },
	{
		path: AppRoutes.NotificationSettings,
		label: 'Уведомления',
		icon: <SettingsIcon sx={{ fontSize: 18 }} />,
	},
]

const accessesItems: SidebarItem[] = [
	{
		path: AppRoutes.Home,
		label: 'Главная',
		icon: <ChevronsRightIcon sx={{ fontSize: 18, transform: 'rotate(180deg)' }} />,
	},
	{ path: AppRoutes.Accesses, label: 'Дашборд', icon: <LayoutDashboardIcon sx={{ fontSize: 18 }} /> },
	{
		path: AppRoutes.Realms,
		label: 'Области',
		icon: <NetworkIcon sx={{ fontSize: 22 }} />,
	},
	{
		path: AppRoutes.UserAccess,
		label: 'Пользователи',
		icon: <UserIcon sx={{ fontSize: 22 }} />,
	},
	{ path: AppRoutes.RoleAccess, label: 'Роли', icon: <ShieldCheckIcon sx={{ fontSize: 22 }} /> },
	{ path: AppRoutes.Permissions, label: 'Права доступа', icon: <KeyRoundIcon fontSize='small' /> },
]

export const sidebarRules: SidebarRule[] = [
	{
		match: path =>
			[
				AppRoutes.Home,
				AppRoutes.Tasks,
				AppRoutes.Sites,
				AppRoutes.Groups,
				AppRoutes.Categories,
				AppRoutes.NotificationSettings,
				'/history',
				'/favorites',
			].includes(path),
		config: { items: homeItems },
	},
	{
		match: path => path.startsWith(AppRoutes.Accesses),
		config: { items: accessesItems },
	},
]
