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
import { ChevronDownIcon, FileUpIcon, PackageOpenIcon, PlusIcon, PrinterIcon } from 'lucide-mui'

import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { pluralize } from '@/utils/plural'
import { useGetEquipmentStatsQuery } from '../equipmentApiSlice'

type Props = {
	onCreate: (kit: boolean) => void
}

export const Header: FC<Props> = ({ onCreate }) => {
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Equipment.Write] })
	const { palette } = useTheme()

	const { data } = useGetEquipmentStatsQuery()
	const stats = data?.data

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
							<PrinterIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Оборудование
					</MenuItem>
					<MenuItem onClick={() => onCreate(true)}>
						<ListItemIcon>
							<PackageOpenIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Комплект
					</MenuItem>
					<MenuItem disabled>
						<ListItemIcon>
							<FileUpIcon sx={{ fontSize: 16, color: '#0e0e0e' }} />
						</ListItemIcon>
						Импорт
					</MenuItem>
				</MenuList>
			</Popover>
		</Box>
	)
}
