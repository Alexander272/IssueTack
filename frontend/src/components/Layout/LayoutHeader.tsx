import { AppBar, Divider, Stack, Toolbar, Tooltip, useTheme } from '@mui/material'
import { Link } from 'react-router'
import Logo from '@/assets/logo.webp'

import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermission } from '@/features/user/hooks/check'
import { useAppSelector } from '@/hooks/redux'
import { useSignOutMutation } from '@/features/auth/authApiSlice'
import { getToken } from '@/features/user/userSlice'
import { AddFileIcon } from '../Icons/AddFileIcon'
import { SearchIcon } from '../Icons/SearchIcon'
import { LogoutIcon } from '../Icons/LogoutIcon'
import { ShieldIcon } from '../Icons/ShieldIcon'
import { ReportsIcon } from '../Icons/ReportsIcon'
import { LedgerIcon } from '../Icons/LedgerIcon'
import { NavBox } from './NavBox'

export const LayoutHeader = () => {
	const { palette } = useTheme()

	const token = useAppSelector(getToken)

	const [signOut] = useSignOutMutation()

	const logoutHandler = () => {
		void signOut(null)
	}

	// const canSeePrice = useCheckPermission(PermRules.Prices.Read)
	// const canEditSettings = useCheckPermission(PermRules.Users.Write)
	// const canSeeStats = useCheckPermission([
	// 	PermRules.SearchLog.Read,
	// 	PermRules.ActivityLog.Read,
	// 	PermRules.Logins.Read,
	// ])

	return (
		<AppBar position='relative' sx={{ borderRadius: 0, alignItems: 'center' }}>
			<Toolbar sx={{ justifyContent: 'space-between', width: '100%', maxWidth: 'xl' }}>
				<Link to='/' aria-label='home page'>
					<Stack
						sx={{
							height: 50,
							overflow: 'hidden',
							alignItems: 'center',
							justifyContent: 'center',
							img: { height: '100%', width: 'auto' },
						}}
					>
						<img src={Logo} alt='logo' />
					</Stack>
				</Link>

				{token ? (
					<Stack
						direction={'row'}
						spacing={2}
						divider={<Divider orientation='vertical' flexItem variant='middle' />}
						sx={{ alignItems: 'center', ml: 'auto' }}
					>
						<Stack direction={'row'} spacing={0.5}>
							{/* {canEditSettings ? (
								<Link to={AppRoutes.Accesses}>
									<Tooltip title='Настройка доступа' disableInteractive>
										<NavBox sx={{ ':hover': { svg: { stroke: palette.primary.main } } }}>
											<ShieldIcon sx={{ fontSize: 26, transition: '0.3s all ease-in-out' }} />
										</NavBox>
									</Tooltip>
								</Link>
							) : null} */}

							{/* {canSeeStats ? (
								<Link to={AppRoutes.Statistics}>
									<Tooltip title='Статистика' disableInteractive>
										<NavBox sx={{ ':hover': { svg: { stroke: palette.primary.main } } }}>
											<ReportsIcon sx={{ fontSize: 26, transition: '0.3s all ease-in-out' }} />
										</NavBox>
									</Tooltip>
								</Link>
							) : null} */}

							{/* <Link to={AppRoutes.CreateOrder}>
								<Tooltip title='Добавить заказ' disableInteractive>
									<NavBox sx={{ ':hover': { svg: { fill: palette.primary.main } } }}>
										<AddFileIcon fill={'#000'} fontSize={26} transition={'0.3s all ease-in-out'} />
									</NavBox>
								</Tooltip>
							</Link>

							<Link to={AppRoutes.Search}>
								<Tooltip title='Поиск' disableInteractive>
									<NavBox sx={{ ':hover': { svg: { fill: palette.primary.main } } }}>
										<SearchIcon
											sx={{ fontSize: 24, transition: '0.3s all ease-in-out', fill: '#000' }}
										/>
									</NavBox>
								</Tooltip>
							</Link> */}
						</Stack>

						<NavBox onClick={logoutHandler} sx={{ ':hover': { svg: { fill: palette.primary.main } } }}>
							<LogoutIcon fill={'#000'} fontSize={24} transition={'0.3s all ease-in-out'} />
						</NavBox>
					</Stack>
				) : null}
			</Toolbar>
		</AppBar>
	)
}
