import { useState, Suspense } from 'react'
import { Outlet, useLocation } from 'react-router'
import { Box, Stack } from '@mui/material'

import { useAppSelector } from '@/hooks/redux'
import { getIsManager } from '@/features/user/userSlice'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { Fallback } from '@/components/Fallback/Fallback'
import { LayoutHeader } from './LayoutHeader'
import { Sidebar } from './Sidebar'
import { sidebarRules } from './sidebarConf'
import { AppRoutes } from '@/pages/router/routes'

const managerOnlyRoutes = new Set([
	AppRoutes.Groups,
	AppRoutes.Categories,
	AppRoutes.Sites,
	AppRoutes.NotificationSettings,
])

// Право на доступ к админскому разделу: наличие любого из write-пермишенов администрирования.
const adminPermissions = ['user:write', 'role:write', 'realm:write', 'permission:write']

export const Layout = () => {
	const location = useLocation()
	const isManager = useAppSelector(getIsManager)
	const isAdminAccess = useCheckPermissions({ anyOf: adminPermissions })
	const sidebarConfig = sidebarRules.find(r => r.match(location.pathname))?.config
	const [mobileOpen, setMobileOpen] = useState(false)

	const filteredConfig = sidebarConfig
		? {
				...sidebarConfig,
				items: sidebarConfig.items.filter(item => {
					if (!isManager && managerOnlyRoutes.has(item.path as '/groups')) return false
					if (location.pathname.startsWith(AppRoutes.Accesses) && !isAdminAccess) return false
					return true
				}),
			}
		: undefined

	return (
		<Box sx={{ minHeight: '100vh', height: '100vh', display: 'flex', flexDirection: 'column', pb: 4 }}>
			<LayoutHeader onMenuClick={() => setMobileOpen(v => !v)} />

			<Stack direction='row' sx={{ flexGrow: 1, overflow: 'hidden' }}>
				{filteredConfig && (
					<Sidebar
						config={filteredConfig}
						mobileOpen={mobileOpen}
						onMobileClose={() => setMobileOpen(false)}
					/>
				)}

				<Suspense key={location.key} fallback={<Fallback />}>
					<Outlet />
				</Suspense>
			</Stack>
		</Box>
	)
}

export default Layout
