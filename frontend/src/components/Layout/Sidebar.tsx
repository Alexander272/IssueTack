import { useState } from 'react'
import { Box, Drawer, List, ListItem, ListItemButton, ListItemIcon, ListItemText, Toolbar, useMediaQuery, useTheme } from '@mui/material'
import { useLocation, useNavigate } from 'react-router'

import type { SidebarConfig } from './sidebarConf'
import { LeftArrowIcon } from '@/components/Icons/LeftArrowIcon'
import { LogoutIcon } from '@/components/Icons/LogoutIcon'
import { useSignOutMutation } from '@/features/auth/authApiSlice'

const COLLAPSED_WIDTH = 60
const EXPANDED_WIDTH = 240

interface SidebarProps {
	config: SidebarConfig
	mobileOpen: boolean
	onMobileClose: () => void
}

export const Sidebar = ({ config, mobileOpen, onMobileClose }: SidebarProps) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('md'))

	const [collapsed, setCollapsed] = useState(() => {
		return localStorage.getItem('sidebarCollapsed') === 'true'
	})

	const handleToggle = () => {
		setCollapsed(prev => {
			const next = !prev
			localStorage.setItem('sidebarCollapsed', String(next))
			return next
		})
	}

	const { items } = config
	const location = useLocation()
	const navigate = useNavigate()
	const [signOut] = useSignOutMutation()

	const handleSwitch = (path: string) => {
		navigate(path)
		if (isMobile) onMobileClose()
	}

	if (isMobile) {
		return (
			<Drawer
				variant='temporary'
				open={mobileOpen}
				onClose={onMobileClose}
				ModalProps={{ keepMounted: true }}
				sx={{
					display: { xs: 'block', md: 'none' },
					'& .MuiDrawer-paper': {
						width: EXPANDED_WIDTH,
						boxSizing: 'border-box',
						px: 2,
						py: 1,
					},
				}}
			>
				<Toolbar />
				<Box sx={{ overflow: 'auto', flexGrow: 1 }}>
					<List sx={{ flexGrow: 1 }}>
						{items.map(item => (
							<ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
								<ListItemButton
									selected={location.pathname === item.path}
									onClick={() => handleSwitch(item.path)}
									sx={{
										borderRadius: '8px',
										px: 2,
										'&.Mui-selected': {
											backgroundColor: 'rgba(25, 118, 210, 0.08)',
											color: 'primary.main',
											svg: { fill: theme => theme.palette.primary.main },
											'& .MuiListItemIcon-root': {
												color: 'primary.main',
												svg: { fill: 'primary.main' },
											},
										},
									}}
								>
									<ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
									<ListItemText primary={item.label} sx={{ fontSize: '14px', fontWeight: 500 }} />
								</ListItemButton>
							</ListItem>
						))}
					</List>
				</Box>

				<Box sx={{ borderTop: '1px solid rgba(0, 0, 0, 0.12)', py: 1 }}>
					<ListItemButton
						onClick={() => signOut(null)}
						sx={{ borderRadius: '8px', px: 2 }}
					>
						<ListItemIcon sx={{ minWidth: 40 }}>
							<LogoutIcon sx={{ fontSize: 20 }} />
						</ListItemIcon>
						<ListItemText primary='Выйти' sx={{ fontSize: '14px', fontWeight: 500 }} />
					</ListItemButton>
				</Box>
			</Drawer>
		)
	}

	const drawerWidth = collapsed ? COLLAPSED_WIDTH : EXPANDED_WIDTH

	return (
		<Drawer
			variant='permanent'
			component='aside'
			sx={{
				width: drawerWidth,
				flexShrink: 0,
				transition: 'width 0.3s ease',
				border: '1px solid rgba(0, 0, 0, 0.12)',
				display: { xs: 'none', md: 'block' },
				[`& .MuiDrawer-paper`]: {
					width: drawerWidth,
					boxSizing: 'border-box',
					paddingX: collapsed ? 0.5 : 2,
					paddingY: 1,
					transition: 'width 0.3s ease',
					overflowX: 'hidden',
				},
			}}
		>
			<Toolbar />
			<Box sx={{ overflow: 'auto', flexGrow: 1 }}>
				<List sx={{ flexGrow: 1 }}>
					{items.map(item => (
						<ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
							<ListItemButton
								selected={location.pathname === item.path}
								onClick={() => handleSwitch(item.path)}
								sx={{
									borderRadius: '8px',
									justifyContent: collapsed ? 'center' : 'flex-start',
									px: collapsed ? 1 : 2,
									'&.Mui-selected': {
										backgroundColor: 'rgba(25, 118, 210, 0.08)',
										color: 'primary.main',
										svg: { fill: theme => theme.palette.primary.main },
										'& .MuiListItemIcon-root': {
											color: 'primary.main',
											svg: { fill: 'primary.main' },
										},
									},
								}}
							>
								<ListItemIcon sx={{ minWidth: collapsed ? 0 : 40 }}>{item.icon}</ListItemIcon>
								{!collapsed && (
									<ListItemText primary={item.label} sx={{ fontSize: '14px', fontWeight: 500 }} />
								)}
							</ListItemButton>
						</ListItem>
					))}
				</List>
			</Box>
			<Box sx={{ borderTop: '1px solid rgba(0, 0, 0, 0.12)', py: 1 }}>
				<ListItemButton
					onClick={handleToggle}
					sx={{
						borderRadius: '8px',
						justifyContent: collapsed ? 'center' : 'flex-start',
						px: collapsed ? 1 : 2,
					}}
				>
					<ListItemIcon sx={{ minWidth: collapsed ? 0 : 40 }}>
						<LeftArrowIcon
							sx={{
								fontSize: 16,
								transform: collapsed ? 'rotate(180deg)' : 'none',
								transition: 'transform 0.3s ease',
							}}
						/>
					</ListItemIcon>
					{!collapsed && <ListItemText primary='Свернуть' sx={{ fontSize: '14px', fontWeight: 500 }} />}
				</ListItemButton>
			</Box>
		</Drawer>
	)
}
