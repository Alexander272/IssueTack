import { useState } from 'react'
import { Box, Drawer, List, ListItem, ListItemButton, ListItemIcon, ListItemText, Toolbar } from '@mui/material'
import { useLocation, useNavigate } from 'react-router'

import type { SidebarConfig } from './sidebarConf'
import { LeftArrowIcon } from '@/components/Icons/LeftArrowIcon'

const COLLAPSED_WIDTH = 60
const EXPANDED_WIDTH = 240

interface SidebarProps {
	config: SidebarConfig
}

export const Sidebar = ({ config }: SidebarProps) => {
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

	const drawerWidth = collapsed ? COLLAPSED_WIDTH : EXPANDED_WIDTH
	const { items } = config
	const location = useLocation()
	const navigate = useNavigate()

	const handleSwitch = (path: string) => {
		navigate(path)
	}

	return (
		<Drawer
			variant='permanent'
			component='aside'
			sx={{
				width: drawerWidth,
				flexShrink: 0,
				transition: 'width 0.3s ease',
				border: '1px solid rgba(0, 0, 0, 0.12)',
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
