import { useState, Fragment, type ReactNode } from 'react'
import {
	Box,
	Divider,
	Drawer,
	List,
	ListItem,
	ListItemButton,
	ListItemIcon,
	ListItemText,
	Toolbar,
	Tooltip,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Link, useLocation, useNavigate } from 'react-router'
import { ArrowLeftIcon, LogOutIcon, ShieldIcon } from 'lucide-mui'

import Logo from '@/assets/logo.webp'
import LogoMini from '@/assets/logo192.webp'
import type { SidebarConfig } from './sidebarConf'
import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { useCan } from '@/features/access/utils/can'
import { useSignOutMutation } from '@/features/auth/authApiSlice'
import { ActiveRealm } from '@/features/realms/components/ActiveRealm'

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

	const compact = isMobile ? false : collapsed

	const { items } = config
	const location = useLocation()
	const navigate = useNavigate()
	const [signOut] = useSignOutMutation()
	const canEditSettings = useCan(PermRules.Users.Write)
	const inAccesses = location.pathname.startsWith(AppRoutes.Accesses)

	const handleSwitch = (path: string) => {
		navigate(path)
		if (isMobile) onMobileClose()
	}

	const renderRealm = () => (
		<Box
			sx={{
				px: compact ? 1 : 2,
				py: 0.5,
				display: 'flex',
				justifyContent: compact ? 'center' : 'flex-start',
			}}
		>
			<ActiveRealm collapsed={compact} />
		</Box>
	)

	const renderButton = ({ icon, label, onClick }: { icon: ReactNode; label: string; onClick: () => void }) => {
		const button = (
			<ListItemButton
				onClick={onClick}
				sx={{
					borderRadius: '8px',
					justifyContent: compact ? 'center' : 'flex-start',
					px: compact ? 1 : 2,
				}}
			>
				<ListItemIcon sx={{ minWidth: compact ? 0 : 40 }}>{icon}</ListItemIcon>
				{!compact && <ListItemText primary={label} sx={{ fontSize: '14px', fontWeight: 500 }} />}
			</ListItemButton>
		)
		if (!compact) return button
		return (
			<Tooltip title={label} placement='right'>
				{button}
			</Tooltip>
		)
	}

	const collapseBlock = (
		<Box sx={{ borderTop: '1px solid rgba(0, 0, 0, 0.12)', py: 1 }}>
			{renderButton({
				icon: (
					<ArrowLeftIcon
						sx={{
							fontSize: 16,
							transform: compact ? 'rotate(180deg)' : 'none',
							transition: 'transform 0.3s ease',
						}}
					/>
				),
				label: compact ? 'Развернуть' : 'Свернуть',
				onClick: handleToggle,
			})}
		</Box>
	)

	const renderNavList = () => (
		<List sx={{ flexGrow: 1 }}>
			{items.map(item => (
				<Fragment key={item.path}>
					{item.divider && <Divider component='li' sx={{ my: 1 }} />}
					<ListItem disablePadding sx={{ mb: 0.5 }}>
						{compact ? (
							<Tooltip title={item.label} placement='right'>
								<ListItemButton
									selected={location.pathname === item.path}
									onClick={() => handleSwitch(item.path)}
									sx={{
										borderRadius: '8px',
										justifyContent: 'center',
										px: 1,
										'&.Mui-selected': {
											backgroundColor: 'rgba(25, 118, 210, 0.08)',
											color: 'primary.main',
											svg: { color: theme => theme.palette.primary.main },
											'& .MuiListItemIcon-root': {
												color: 'primary.main',
											},
										},
									}}
								>
									<ListItemIcon sx={{ minWidth: 0 }}>{item.icon}</ListItemIcon>
								</ListItemButton>
							</Tooltip>
						) : (
							<ListItemButton
								selected={location.pathname === item.path}
								onClick={() => handleSwitch(item.path)}
								sx={{
									borderRadius: '8px',
									justifyContent: 'flex-start',
									px: 2,
									'&.Mui-selected': {
										backgroundColor: 'rgba(25, 118, 210, 0.08)',
										color: 'primary.main',
										svg: { color: theme => theme.palette.primary.main },
										'& .MuiListItemIcon-root': {
											color: 'primary.main',
										},
									},
								}}
							>
								<ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
								<ListItemText primary={item.label} sx={{ fontSize: '14px', fontWeight: 500 }} />
							</ListItemButton>
						)}
					</ListItem>
				</Fragment>
			))}

			{canEditSettings && !inAccesses && (
				<ListItem key='access-settings' disablePadding sx={{ mb: 0.5 }}>
					{renderButton({
						icon: <ShieldIcon sx={{ fontSize: 20 }} />,
						label: 'Доступ',
						onClick: () => navigate(AppRoutes.Accesses),
					})}
				</ListItem>
			)}
			<ListItem key='log-out' disablePadding sx={{ mb: 0.5 }}>
				{renderButton({
					icon: <LogOutIcon sx={{ fontSize: 20 }} />,
					label: 'Выйти',
					onClick: () => signOut(null),
				})}
			</ListItem>
		</List>
	)

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
				{renderRealm()}
				<Box sx={{ overflow: 'auto', flexGrow: 1 }}>{renderNavList()}</Box>
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
			<Box
				component={Link}
				to='/'
				aria-label='home page'
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					overflow: 'hidden',
					p: collapsed ? 0.5 : 1.5,
					pt: collapsed ? 1 : 1.5,
					img: { maxWidth: '100%' },
				}}
			>
				<img
					src={collapsed ? LogoMini : Logo}
					alt='logo'
					style={{ maxHeight: collapsed ? 32 : 44, width: 'auto' }}
				/>
			</Box>
			{renderRealm()}
			<Box sx={{ overflow: 'auto', flexGrow: 1 }}>{renderNavList()}</Box>
			{collapseBlock}
		</Drawer>
	)
}
