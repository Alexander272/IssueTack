import { useState, type ChangeEvent, type FC, type MouseEvent } from 'react'
import {
	Badge,
	Box,
	Button,
	Drawer,
	IconButton,
	MenuItem,
	TextField,
	ToggleButton,
	ToggleButtonGroup,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { ChevronsDownIcon, ChevronsUpIcon, FilterIcon, SearchIcon, XIcon } from 'lucide-mui'

import { isTreeMode, type EquipmentFilters, type TypeFilter } from './types'
import { EQUIPMENT_STATUS_OPTIONS } from '../../types/equipment'

type Props = {
	filters: EquipmentFilters
	// Передаем partial-объект для обновления только нужных полей
	onFiltersChange: (filters: Partial<EquipmentFilters>) => void
	// Развернуть/свернуть всё: true — раскрыть, false — свернуть.
	onToggleAll: (expand: boolean) => void
}

// Выносим статичные стили за пределы компонента, чтобы они не пересоздавались при каждом рендере
const boxStyles = {
	bgcolor: 'background.paper',
	p: { xs: 2, sm: 2.5 },
	borderRadius: '12px',
	border: '1px solid',
	borderColor: 'divider',
	mb: 2,
	display: 'flex',
	flexDirection: { xs: 'column', sm: 'row' },
	gap: 1.5,
	flexWrap: 'wrap',
} as const

const buttonStyles = (treeMode: boolean) =>
	({
		flexShrink: 0,
		height: 40,
		borderColor: 'action.disabledBackground',
		textTransform: 'none',
		color: treeMode ? 'text.primary' : 'text.disabled',
		whiteSpace: 'nowrap',
	}) as const

const sheetLabelSx = { fontSize: '0.75rem', fontWeight: 600, color: '#374151', mb: 0.75, display: 'block' } as const

// Тулбар фильтров — в стиле страницы заявок (белая карточка). На широком экране
// фильтры лежат в строке; на узком — главной остаётся строка поиска, а тип, статус и
// «Развернуть/Свернуть всё» уезжают в шторку под кнопкой «Фильтры» (экономит высоту
// экрана и убирает загадочные стрелки из основной строки).
export const EquipmentToolbar: FC<Props> = ({ filters, onFiltersChange, onToggleAll }) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('md'))
	const { search, typeFilter, status } = filters
	const [sheetOpen, setSheetOpen] = useState(false)

	// Режим-дерево вычисляется из поиска: в плоском списке кнопки раскрытия гаснут.
	const treeMode = isTreeMode(search)

	// Активные фильтры для бейджа на кнопке «Фильтры»: не «все» тип и статус.
	const activeCount = (typeFilter !== 'all' ? 1 : 0) + (status !== 'all' ? 1 : 0)

	// Изолированные обработчики для избежания инлайн-функций в рендере
	const handleSearchChange = (e: ChangeEvent<HTMLInputElement>) => {
		onFiltersChange({ search: e.target.value })
	}

	const handleTypeChange = (_e: MouseEvent<HTMLElement>, value: TypeFilter | null) => {
		if (value) onFiltersChange({ typeFilter: value })
	}

	const handleStatusChange = (e: ChangeEvent<HTMLInputElement>) => {
		onFiltersChange({ status: e.target.value })
	}

	const searchField = (
		<TextField
			size='small'
			placeholder='Поиск: название, инв. № или серийник…'
			value={search}
			onChange={handleSearchChange}
			slotProps={{
				input: {
					startAdornment: <SearchIcon sx={{ fontSize: 16, color: 'text.secondary', mr: 1 }} />,
				},
			}}
			sx={{ flex: 1, minWidth: { xs: 0, sm: 250 } }}
		/>
	)

	const typeGroup = (
		<ToggleButtonGroup
			exclusive
			value={typeFilter}
			color='primary'
			onChange={handleTypeChange}
			sx={{ flexShrink: 0, height: 40, width: { xs: '100%', sm: 'auto' } }}
		>
			<ToggleButton value='all' sx={{ textTransform: 'none', px: 2, borderRadius: 2, flexGrow: 1 }}>
				Все
			</ToggleButton>
			<ToggleButton value='unit' sx={{ textTransform: 'none', px: 2, flexGrow: 1 }}>
				Единицы
			</ToggleButton>
			<ToggleButton value='kit' sx={{ textTransform: 'none', px: 2, borderRadius: 2, flexGrow: 1 }}>
				Комплекты
			</ToggleButton>
		</ToggleButtonGroup>
	)

	const statusSelect = (
		<TextField
			select
			size='small'
			value={status}
			onChange={handleStatusChange}
			sx={{ minWidth: 180, flexShrink: 0, width: { xs: '100%', sm: 'auto' } }}
		>
			<MenuItem value='all'>Все статусы</MenuItem>
			{EQUIPMENT_STATUS_OPTIONS.map(s => (
				<MenuItem key={s.value} value={s.value}>
					{s.label}
				</MenuItem>
			))}
		</TextField>
	)

	if (isMobile) {
		return (
			<>
				<Box sx={boxStyles}>
					{searchField}

					<Badge badgeContent={activeCount} color='primary' invisible={activeCount === 0}>
						<Button
							variant='outlined'
							color='inherit'
							startIcon={<FilterIcon sx={{ fontSize: 18 }} />}
							onClick={() => setSheetOpen(true)}
							sx={{
								flexShrink: 0,
								flexGrow: 1,
								height: 40,
								textTransform: 'none',
								whiteSpace: 'nowrap',
								borderColor: '#6c6c6c',
							}}
						>
							Фильтры
						</Button>
					</Badge>
				</Box>

				<Drawer
					anchor='bottom'
					open={sheetOpen}
					onClose={() => setSheetOpen(false)}
					slotProps={{
						paper: {
							sx: {
								borderTopLeftRadius: 16,
								borderTopRightRadius: 16,
								px: 2,
								pt: 1.5,
								pb: 3,
								maxHeight: '85vh',
							},
						},
					}}
				>
					{/* «Ручка» шторки — визуальный признак, что её можно тянуть/закрыть. */}
					<Box sx={{ width: 40, height: 4, borderRadius: 999, bgcolor: '#e5e7eb', mx: 'auto', mb: 1.5 }} />

					<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
						<Typography sx={{ fontWeight: 700, fontSize: '1rem', color: '#1f2937' }}>Фильтры</Typography>
						<IconButton onClick={() => setSheetOpen(false)} size='small' aria-label='Закрыть'>
							<XIcon sx={{ fontSize: 18 }} />
						</IconButton>
					</Box>

					<Typography sx={sheetLabelSx}>Тип</Typography>
					{typeGroup}

					<Box sx={{ mt: 2 }}>
						<Typography sx={sheetLabelSx}>Статус</Typography>
						{statusSelect}
					</Box>

					<Button
						variant='contained'
						fullWidth
						onClick={() => setSheetOpen(false)}
						sx={{ mt: 3, textTransform: 'none' }}
					>
						Готово
					</Button>
				</Drawer>
			</>
		)
	}

	return (
		<Box sx={boxStyles}>
			{searchField}
			{typeGroup}

			<Button
				variant='outlined'
				color='inherit'
				disabled={!treeMode}
				aria-label='Развернуть всё'
				onClick={() => onToggleAll(true)}
				sx={buttonStyles(treeMode)}
			>
				<ChevronsDownIcon sx={{ fontSize: 16, mr: 1 }} />
				Развернуть всё
			</Button>

			<Button
				variant='outlined'
				color='inherit'
				disabled={!treeMode}
				aria-label='Свернуть всё'
				onClick={() => onToggleAll(false)}
				sx={buttonStyles(treeMode)}
			>
				<ChevronsUpIcon sx={{ fontSize: 16, mr: 1 }} />
				Свернуть всё
			</Button>

			{statusSelect}
		</Box>
	)
}
