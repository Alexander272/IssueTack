import type { FC, ReactElement } from 'react'
import {
	Dialog,
	DialogContent,
	DialogTitle,
	IconButton,
	Typography,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Box,
} from '@mui/material'
import {
	Computer,
	PhoneAndroid,
	TabletAndroid,
	SmartToy,
	CheckCircle,
	Cancel,
	Public,
	SettingsApplications,
} from '@mui/icons-material'

import type { IUserData } from '@/features/user/types/user'
import { getSmartDate } from '@/utils/date'
import { useGetUserLoginsQuery } from '@/features/user/usersApiSlice'
import { TimesIcon } from '@/components/Icons/TimesIcon'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { ChromeIcon } from '@/components/Icons/Browsers/ChromeIcon'
import { FirefoxIcon } from '@/components/Icons/Browsers/FirefoxIcon'
import { SafariIcon } from '@/components/Icons/Browsers/SafariIcon'
import { ExplorerIcon } from '@/components/Icons/Browsers/ExplorerIcon'
import { OperaIcon } from '@/components/Icons/Browsers/OperaIcon'
import { GlobeIcon } from '@/components/Icons/Browsers/GlobeIcon'
import { WindowsIcon } from '@/components/Icons/Systems/WindowsIcon'
import { AppleIcon } from '@/components/Icons/Systems/AppleIcon'
import { IphoneIcon } from '@/components/Icons/Systems/IphoneIcon'
import { AndroidIcon } from '@/components/Icons/Systems/AndroidIcon'
import { LinuxIcon } from '@/components/Icons/Systems/LinuxIcon'
import { UbuntuIcon } from '@/components/Icons/Systems/UbuntuIcon'

type Props = {
	user: IUserData | null
	onClose: () => void
}

const getDeviceIcon = (device: string): ReactElement => {
	switch (device) {
		case 'desktop':
			return <Computer sx={{ fontSize: 14 }} />
		case 'mobile':
			return <PhoneAndroid sx={{ fontSize: 14 }} />
		case 'tablet':
			return <TabletAndroid sx={{ fontSize: 14 }} />
		default:
			return <SmartToy sx={{ fontSize: 14 }} />
	}
}

const getDeviceLabel = (device: string): string => {
	switch (device) {
		case 'desktop':
			return 'ПК'
		case 'mobile':
			return 'Телефон'
		case 'tablet':
			return 'Планшет'
		default:
			return device
	}
}

const browserConfig: Record<string, { icon: ReactElement; colors: { bg: string; color: string } }> = {
	Chrome: {
		icon: <ChromeIcon sx={{ fontSize: 14, fill: '#2563eb' }} />,
		colors: { bg: '#eff6ff', color: '#2563eb' },
	},
	Firefox: {
		icon: <FirefoxIcon sx={{ fontSize: 14, fill: '#ea580c' }} />,
		colors: { bg: '#fff7ed', color: '#ea580c' },
	},
	Safari: {
		icon: <SafariIcon sx={{ fontSize: 14, fill: '#2563eb' }} />,
		colors: { bg: '#eff6ff', color: '#2563eb' },
	},
	Edge: {
		icon: <ExplorerIcon sx={{ fontSize: 14, fill: '#2563eb' }} />,
		colors: { bg: '#eff6ff', color: '#2563eb' },
	},
	Opera: { icon: <OperaIcon sx={{ fontSize: 14, fill: '#ea260c' }} />, colors: { bg: '#fff7ed', color: '#ea260c' } },
	Yandex: { icon: <GlobeIcon sx={{ fontSize: 14, fill: '#ca8a04' }} />, colors: { bg: '#fef9c3', color: '#ca8a04' } },
}

const getBrowserConfig = (browser: string) => {
	const config = browserConfig[browser]
	if (config) return config
	return {
		icon: <GlobeIcon sx={{ fontSize: 14 }} />,
		colors: { bg: '#f8fafc', color: '#475569' },
	}
}

const osConfig: Record<string, { icon: ReactElement; colors: { bg: string; color: string } }> = {
	Windows: {
		icon: <WindowsIcon sx={{ fontSize: 14, fill: '#2563eb' }} />,
		colors: { bg: '#eff6ff', color: '#2563eb' },
	},
	Mac: { icon: <AppleIcon sx={{ fontSize: 14, fill: '#b45309' }} />, colors: { bg: '#eff6ff', color: '#2563eb' } },
	iOS: { icon: <IphoneIcon sx={{ fontSize: 14, fill: '#b45309' }} />, colors: { bg: '#eff6ff', color: '#2563eb' } },
	Android: {
		icon: <AndroidIcon sx={{ fontSize: 14, fill: '#2563eb' }} />,
		colors: { bg: '#ecfdf5', color: '#047857' },
	},
	Linux: { icon: <LinuxIcon sx={{ fontSize: 14, fill: '#ca8a04' }} />, colors: { bg: '#fef9c3', color: '#ca8a04' } },
	Ubuntu: {
		icon: <UbuntuIcon sx={{ fontSize: 14, fill: '#ca8a04' }} />,
		colors: { bg: '#fef9c3', color: '#ca8a04' },
	},
}

const getOsConfig = (os: string) => {
	const config = osConfig[os]
	if (config) return config
	return {
		icon: <SettingsApplications sx={{ fontSize: 14 }} />,
		colors: { bg: '#f8fafc', color: '#475569' },
	}
}

const chipColors = {
	success: { bg: '#ecfdf5', color: '#047857', dot: '#10b981' },
	error: { bg: '#fef2f2', color: '#b91c1c', dot: '#ef4444' },
	info: { bg: '#eff6ff', color: '#1d4ed8', dot: '#3b82f6' },
	warning: { bg: '#fffbeb', color: '#b45309', dot: '#f59e0b' },
	default: { bg: '#f8fafc', color: '#475569', dot: '#94a3b8' },
}

type ChipVariant = keyof typeof chipColors

const Chip = ({
	variant = 'default',
	colors,
	label,
	icon,
}: {
	variant?: ChipVariant
	colors?: { bg: string; color: string }
	label: string
	icon: ReactElement
}) => {
	const theme = colors || chipColors[variant]

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				gap: 1,
				bgcolor: theme.bg,
				color: theme.color,
				px: 2,
				py: 0.75,
				borderRadius: '16px',
				fontSize: '0.875rem',
				fontWeight: 500,
			}}
		>
			{icon}
			{label}
		</Box>
	)
}

export const LoginsModal: FC<Props> = ({ user, onClose }) => {
	const { data, isFetching } = useGetUserLoginsQuery(user?.ssoId || '', { skip: !user })

	return (
		<Dialog
			open={Boolean(user)}
			onClose={onClose}
			fullWidth
			maxWidth='lg'
			slotProps={{
				paper: {
					sx: {
						borderRadius: '16px',
						p: 1,
					},
				},
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Данные о входах
				</Typography>

				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			{isFetching && <BoxFallback />}

			<DialogContent>
				{data?.data && data.data.length > 0 ? (
					<TableContainer>
						<Table size='small'>
							<TableHead>
								<TableRow sx={{ background: 'action.hover' }}>
									<TableCell sx={{ fontWeight: 700 }}>Вход</TableCell>
									<TableCell sx={{ fontWeight: 700, textAlign: 'center' }}>
										Последняя активность
									</TableCell>
									{/* //TODO добавить event */}
									<TableCell sx={{ fontWeight: 700, textAlign: 'center' }}>IP адрес</TableCell>
									<TableCell sx={{ fontWeight: 700, textAlign: 'center' }}>Устройство</TableCell>
									<TableCell sx={{ fontWeight: 700, textAlign: 'center' }}>Бот</TableCell>
									<TableCell sx={{ fontWeight: 700, textAlign: 'center' }}>Успешно</TableCell>
								</TableRow>
							</TableHead>
							<TableBody>
								{data.data.map(login => {
									const metadata = login.metadata
									const browser = metadata?.browser
									const device = metadata?.device
									const os = metadata?.os
									const isBot = metadata?.isBot
									const success = metadata?.success

									return (
										<TableRow key={login.id} hover>
											<TableCell>
												<Typography sx={{ color: 'text.secondary' }}>
													{getSmartDate(login.loginAt)}
												</Typography>
											</TableCell>
											<TableCell align='center'>
												<Typography sx={{ color: 'text.secondary' }}>
													{getSmartDate(login.lastActivityAt)}
												</Typography>
											</TableCell>
											<TableCell align='center'>
												<Chip
													variant='info'
													label={login.ipAddress || '-'}
													icon={<Public sx={{ fontSize: 14 }} />}
												/>
											</TableCell>
											<TableCell align='center'>
												{browser || device ? (
													<Box sx={{ display: 'flex', gap: 0.5, justifyContent: 'center' }}>
														{browser && (
															<Chip
																colors={getBrowserConfig(browser).colors}
																label={browser}
																icon={getBrowserConfig(browser).icon}
															/>
														)}
														{device && (
															<Chip
																colors={{ bg: '#f8fafc', color: '#475569' }}
																label={getDeviceLabel(device)}
																icon={getDeviceIcon(device)}
															/>
														)}
														{os && (
															<Chip
																colors={getOsConfig(os).colors}
																label={os}
																icon={getOsConfig(os).icon}
															/>
														)}
													</Box>
												) : (
													<Typography sx={{ color: 'text.disabled' }}>-</Typography>
												)}
											</TableCell>
											<TableCell align='center'>
												{isBot !== undefined ? (
													<Chip
														variant={isBot ? 'error' : 'success'}
														label={isBot ? 'Да' : 'Нет'}
														icon={<SmartToy sx={{ fontSize: 14 }} />}
													/>
												) : (
													<Typography sx={{ color: 'text.disabled' }}>-</Typography>
												)}
											</TableCell>
											<TableCell align='center'>
												{success !== undefined ? (
													<Chip
														variant={success ? 'success' : 'error'}
														label={success ? 'Да' : 'Нет'}
														icon={
															success ? (
																<CheckCircle sx={{ fontSize: 14 }} />
															) : (
																<Cancel sx={{ fontSize: 14 }} />
															)
														}
													/>
												) : (
													<Typography sx={{ color: 'text.disabled' }}>-</Typography>
												)}
											</TableCell>
										</TableRow>
									)
								})}
							</TableBody>
						</Table>
					</TableContainer>
				) : (
					<Typography sx={{ color: 'text.secondary', textAlign: 'center', py: 3 }}>
						Нет данных о входах
					</Typography>
				)}
			</DialogContent>
		</Dialog>
	)
}
