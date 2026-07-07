import { AppBar, Divider, IconButton, Stack, Toolbar, Tooltip, useTheme } from '@mui/material'
import { LogOutIcon, ShieldIcon } from 'lucide-mui'
import { Link } from 'react-router'
import Logo from '@/assets/logo.webp'

import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { useCan } from '@/features/access/utils/can'
import { useAppSelector } from '@/hooks/redux'
import { useSignOutMutation } from '@/features/auth/authApiSlice'
import { getToken, getUserRealms } from '@/features/user/userSlice'
import { ActiveRealm } from '@/features/realms/components/ActiveRealm'
import { NavBox } from './NavBox'

interface LayoutHeaderProps {
	onMenuClick?: () => void
}

export const LayoutHeader = ({ onMenuClick }: LayoutHeaderProps) => {
	const { palette } = useTheme()

	const token = useAppSelector(getToken)
	const realms = useAppSelector(getUserRealms)

	const [signOut] = useSignOutMutation()

	const logoutHandler = () => {
		void signOut(null)
	}

	const canEditSettings = useCan(PermRules.Users.Write)
	// const canSeeStats = useCheckPermission([
	// 	PermRules.SearchLog.Read,
	// 	PermRules.ActivityLog.Read,
	// 	PermRules.Logins.Read,
	// ])

	return (
		<AppBar
			position='relative'
			sx={{ borderRadius: 0, alignItems: 'center', zIndex: theme => theme.zIndex.drawer + 1 }}
		>
			<Toolbar sx={{ justifyContent: 'space-between', width: '100%', maxWidth: 'xl' }}>
				<IconButton
					onClick={onMenuClick}
					sx={{ display: { md: 'none' }, mr: 1, color: 'text.primary', fontSize: '1.5rem' }}
				>
					☰
				</IconButton>

				<Stack
					component={Link}
					to='/'
					aria-label='home page'
					sx={{
						height: 50,
						overflow: 'hidden',
						alignItems: 'center',
						justifyContent: 'center',
						img: { height: '100%', width: 'auto' },
						position: { xs: 'absolute', md: 'static' },
						left: '50%',
						transform: { xs: 'translateX(-50%)', md: 'none' },
					}}
				>
					<img src={Logo} alt='logo' />
				</Stack>

				{token ? (
					<Stack
						direction={'row'}
						spacing={2}
						divider={<Divider orientation='vertical' flexItem variant='middle' />}
						sx={{ alignItems: 'center', ml: 'auto' }}
					>
						{(realms?.length || 0) > 1 && (
							<NavBox>
								<ActiveRealm />
							</NavBox>
						)}

						{/* <Stack direction={'row'} spacing={0.5}> */}
						{canEditSettings ? (
							<Link to={AppRoutes.Accesses}>
								<Tooltip title='Настройка доступа' disableInteractive>
									<NavBox sx={{ ':hover': { svg: { color: palette.primary.main } } }}>
										<ShieldIcon
											sx={{ color: '#404040', fontSize: 26, transition: '0.3s all ease-in-out' }}
										/>
									</NavBox>
								</Tooltip>
							</Link>
						) : null}

						{/* {canSeeStats ? (
								<Link to={AppRoutes.Statistics}>
									<Tooltip title='Статистика' disableInteractive>
										<NavBox sx={{ ':hover': { svg: { stroke: palette.primary.main } } }}>
											<ReportsIcon sx={{ fontSize: 26, transition: '0.3s all ease-in-out' }} />
										</NavBox>
									</Tooltip>
								</Link>
							) : null} */}
						{/* </Stack> */}

						<NavBox
							onClick={logoutHandler}
							sx={{
								display: { xs: 'none', md: 'flex' },
								':hover': { svg: { color: palette.primary.main } },
							}}
						>
							<LogOutIcon sx={{ color: '#404040', fontSize: 24, transition: '0.3s all ease-in-out' }} />
						</NavBox>
					</Stack>
				) : null}
			</Toolbar>
		</AppBar>
	)
}
