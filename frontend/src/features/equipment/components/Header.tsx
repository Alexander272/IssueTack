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

	const { data } = useGetEquipmentStatsQuery()
	const stats = data?.data

	const openSetting = () => navigate(AppRoutes.EquipmentSettings)

	const [anchor, setAnchor] = useState<HTMLButtonElement | null>(null)
	const open = Boolean(anchor)

	return (
		<Box
			sx={{
				display: 'flex',
				flexDirection: { xs: 'column', sm: 'row' },
				justifyContent: 'space-between',
				alignItems: { xs: 'flex-start', sm: 'center' },
				gap: { xs: 2, sm: 0 },
				mb: 2,
			}}
		>
			<Box>
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

			{canWrite && (
				<Stack direction='row' spacing={1} sx={{ flexWrap: 'wrap', gap: 1, width: { xs: '100%', sm: 'auto' } }}>
					<Button variant='text' onClick={openSetting}>
						<SettingIcon sx={{ fontSize: 22 }} />
					</Button>

					<Button
						variant='outlined'
						color='inherit'
						onClick={() => onCreate(false)}
						sx={{ textTransform: 'none', background: '#fff', borderColor: '#6c6c6c' }}
					>
						<PackageOpenIcon sx={{ fontSize: 16, mr: 1 }} />
						Новый комплект
					</Button>

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
					<MenuItem onClick={() => onCreate(false)}>
						<ListItemIcon>
							<MonitorIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Вручную
					</MenuItem>
					{/* <MenuItem onClick={() => onCreate(true)}>
						<ListItemIcon>
							<PackageOpenIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Комплект
					</MenuItem> */}
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
