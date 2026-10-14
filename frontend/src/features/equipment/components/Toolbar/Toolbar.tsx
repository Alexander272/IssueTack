import type { ChangeEvent, FC, MouseEvent } from 'react'
import { Box, Button, MenuItem, TextField, ToggleButton, ToggleButtonGroup } from '@mui/material'
import { ChevronsDownIcon, ChevronsUpIcon, SearchIcon } from 'lucide-mui'

import type { EquipmentFilters, TypeFilter } from './types'
import { EQUIPMENT_STATUS_OPTIONS } from '../../types/equipment'

type Props = {
	filters: EquipmentFilters
	// Передаем partial-объект для обновления только нужных полей
	onFiltersChange: (filters: Partial<EquipmentFilters>) => void
	treeMode: boolean
	onExpandAll: () => void
	onCollapseAll: () => void
}

// Выносим статичные стили за пределы компонента, чтобы они не пересоздавались при каждом рендере
const boxStyles = {
	bgcolor: 'background.paper',
	p: 2.5,
	borderRadius: '12px',
	border: '1px solid',
	borderColor: 'divider',
	mb: 2,
	display: 'flex',
	alignItems: 'center',
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

// Тулбар фильтров — в стиле страницы заявок (белая карточка).
export const EquipmentToolbar: FC<Props> = ({ filters, onFiltersChange, treeMode, onExpandAll, onCollapseAll }) => {
	const { search, typeFilter, status } = filters

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

	return (
		<Box sx={boxStyles}>
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
				sx={{ flex: { xs: '1 1 100%', sm: 1 }, minWidth: { xs: 150, sm: 250 } }}
			/>

			<ToggleButtonGroup
				exclusive
				value={typeFilter}
				onChange={handleTypeChange}
				sx={{ flexShrink: 0, height: 40 }}
			>
				<ToggleButton value='all' sx={{ textTransform: 'none', px: 2 }}>
					Все
				</ToggleButton>
				<ToggleButton value='unit' sx={{ textTransform: 'none', px: 2 }}>
					Единицы
				</ToggleButton>
				<ToggleButton value='kit' sx={{ textTransform: 'none', px: 2 }}>
					Комплекты
				</ToggleButton>
			</ToggleButtonGroup>

			<Button
				variant='outlined'
				color='inherit'
				disabled={!treeMode}
				onClick={onExpandAll}
				sx={buttonStyles(treeMode)}
			>
				<ChevronsDownIcon sx={{ fontSize: 16, mr: 1 }} />
				Развернуть всё
			</Button>

			<Button
				variant='outlined'
				color='inherit'
				disabled={!treeMode}
				onClick={onCollapseAll}
				sx={buttonStyles(treeMode)}
			>
				<ChevronsUpIcon sx={{ fontSize: 16, mr: 1 }} />
				Свернуть всё
			</Button>

			<TextField select value={status} onChange={handleStatusChange} sx={{ minWidth: 180, flexShrink: 0 }}>
				<MenuItem value='all'>Все статусы</MenuItem>
				{EQUIPMENT_STATUS_OPTIONS.map(s => (
					<MenuItem key={s.value} value={s.value}>
						{s.label}
					</MenuItem>
				))}
			</TextField>
		</Box>
	)
}
