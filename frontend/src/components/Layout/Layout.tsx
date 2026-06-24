import { useState, Suspense } from 'react'
import { Outlet, useLocation } from 'react-router'
import { Box, Stack } from '@mui/material'

import { Fallback } from '@/components/Fallback/Fallback'
import { LayoutHeader } from './LayoutHeader'
import { Sidebar } from './Sidebar'
import { sidebarRules } from './sidebarConf'

export const Layout = () => {
	const location = useLocation()
	const sidebarConfig = sidebarRules.find(r => r.match(location.pathname))?.config
	const [mobileOpen, setMobileOpen] = useState(false)

	return (
		<Box sx={{ minHeight: '100vh', height: '100vh', display: 'flex', flexDirection: 'column', pb: 4 }}>
			<LayoutHeader onMenuClick={() => setMobileOpen(v => !v)} />

			<Stack direction='row' sx={{ flexGrow: 1, overflow: 'hidden' }}>
				{sidebarConfig && (
					<Sidebar
						config={sidebarConfig}
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
