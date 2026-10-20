import { useState, type FC } from 'react'
import {
	Box,
	Button,
	ButtonGroup,
	MenuList,
	MenuItem,
	Popover,
	Stack,
	Typography,
	useTheme,
	useMediaQuery,
	ListItemIcon,
} from '@mui/material'
import { ChevronDownIcon, FileUpIcon, MonitorIcon, PackageOpenIcon, PlusIcon } from 'lucide-mui'
import { useNavigate } from 'react-router'

import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { useGetEquipmentStatsQuery } from '../equipmentApiSlice'
import { SettingIcon } from '@/components/Icons/SettingIcon'

type Props = {
	onCreate: (kit: boolean) => void
}

export const Header: FC<Props> = ({ onCreate }) => {
	const navigate = useNavigate()
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const { palette } = useTheme()
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))

	const { data } = useGetEquipmentStatsQuery()
	const stats = data?.data

	const openSetting = () => navigate(AppRoutes.EquipmentSettings)

	const [anchor, setAnchor] = useState<HTMLButtonElement | null>(null)
	const open = Boolean(anchor)

	// Одна и та же кнопка настроек в двух местах: на мобильном — в углу рядом с
	// заголовком, на десктопе — в ряду действий с кнопками создания.
	const settingsButton = (
		<Button variant='text' onClick={openSetting} aria-label='Настройки оборудования'>
			<SettingIcon sx={{ fontSize: 22 }} />
		</Button>
	)

	return (
		<Box
			sx={{
				display: 'flex',
				flexDirection: { xs: 'column', sm: 'row' },
				// Кнопки не переносятся «по одной»: если ряд «заголовок + действия»
				// не влезает, вниз уезжает вся группа кнопок и прижимается вправо,
				// а не отдельные кнопки влево. Раньше в диапазонах ~600–800 и
				// ~900–1040 (сайдбар сужает контент) они вставали в два ряда.
				flexWrap: 'wrap',
				justifyContent: 'space-between',
				alignItems: { xs: 'flex-start', sm: 'center' },
				columnGap: { sm: 2 },
				rowGap: { xs: 2, sm: 1 },
				mb: 2,
			}}
		>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'flex-start',
					gap: 1,
					width: { xs: '100%', sm: 'auto' },
					minWidth: 0,
					// Блок заголовка со статистикой не сжимаем: строка переносится
					// в группе действий, а не здесь (иначе ломалась бы подпись).
					flexShrink: 0,
				}}
			>
				<Box sx={{ minWidth: 0 }}>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						Оборудование
					</Typography>
					{stats && (
						<Typography variant='body2' sx={{ color: '#6b7280' }}>
							{stats.units} {pluralize(stats.units, ['единица', 'единицы', 'единиц'])} · {stats.kits}{' '}
							{pluralize(stats.kits, ['комплект', 'комплекта', 'комплектов'])} · {stats.stockParts}{' '}
							{pluralize(stats.stockParts, ['компонент', 'компонента', 'компонентов'])} на складе
						</Typography>
					)}
				</Box>

				{/* На мобильном шестерёнка стоит в углу рядом с заголовком. */}
				{canWrite && isMobile && settingsButton}
			</Box>

			{canWrite && (
				<Stack
					direction='row'
					spacing={1}
					sx={{
						flexWrap: 'nowrap',
						flexShrink: 0,
						gap: 1,
						width: { xs: '100%', sm: 'auto' },
						// Когда группа действий уезжает на свою строку — прижимаем вправо.
						ml: { sm: 'auto' },
					}}
				>
					{!isMobile && settingsButton}

					{/* На десктопе отдельная кнопка «Новый комплект»; на мобильном её роль
					    играет пункт меню у кнопки «Создать». */}
					{!isMobile && (
						<Button
							variant='outlined'
							color='inherit'
							onClick={() => onCreate(true)}
							sx={{ textTransform: 'none', background: '#fff', borderColor: '#6c6c6c' }}
						>
							<PackageOpenIcon sx={{ fontSize: 16, mr: 1 }} />
							Новый комплект
						</Button>
					)}

					<ButtonGroup variant='outlined' sx={{ borderRadius: '8px', width: { xs: '100%', sm: 'auto' } }}>
						<Button
							variant='outlined'
							onClick={() => onCreate(false)}
							sx={{
								textTransform: 'none',
								background: '#fff',
								width: { xs: '100%', sm: 'auto' },
							}}
						>
							<PlusIcon sx={{ color: palette.primary.main, fontSize: 16, mr: 1 }} />
							Создать
						</Button>

						<Button
							variant='outlined'
							aria-controls={open ? 'split-button-menu' : undefined}
							aria-expanded={open ? 'true' : undefined}
							aria-label='select merge strategy'
							aria-haspopup='menu'
							onClick={e => setAnchor(e.currentTarget)}
							sx={{ background: '#fff' }}
						>
							<ChevronDownIcon sx={{ color: palette.primary.main, fontSize: 16 }} />
						</Button>
					</ButtonGroup>
				</Stack>
			)}

			<Popover
				open={open}
				anchorEl={anchor}
				onClose={() => setAnchor(null)}
				anchorOrigin={{
					vertical: 'bottom',
					horizontal: 'right',
				}}
			>
				<MenuList>
					<MenuItem
						onClick={() => {
							setAnchor(null)
							onCreate(false)
						}}
					>
						<ListItemIcon>
							<MonitorIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Оборудование
					</MenuItem>
					<MenuItem
						onClick={() => {
							setAnchor(null)
							onCreate(true)
						}}
					>
						<ListItemIcon>
							<PackageOpenIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Комплект
					</MenuItem>
					<MenuItem disabled>
						<ListItemIcon>
							<FileUpIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Импортировать
					</MenuItem>
				</MenuList>
			</Popover>
		</Box>
	)
}
