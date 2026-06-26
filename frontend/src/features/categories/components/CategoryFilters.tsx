import { useState, type FC } from 'react'
import { Box, Button, MenuItem, Select, TextField, Typography, useMediaQuery, useTheme } from '@mui/material'

import type { IGroup } from '@/features/groups/types/group'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { RefreshIcon } from '@/components/Icons/RefreshIcon'
import { FilterIcon } from '@/components/Icons/FilterIcon'

export interface CategoryFilterState {
	group: string
	status: string
	priority: string
	search: string
}

type Props = {
	groups: IGroup[]
	filters: CategoryFilterState
	onChange: (patch: Partial<CategoryFilterState>) => void
	onReset: () => void
}

export const CategoryFilters: FC<Props> = ({ groups, filters, onChange, onReset }) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('md'))
	const [showFilters, setShowFilters] = useState(!isMobile)

	return (
		<Box
			sx={{
				bgcolor: '#fff',
				p: 2,
				borderRadius: '12px',
				border: '1px solid #e5e7eb',
				mb: 3,

				display: 'flex',
				alignItems: 'center',
				gap: 1,
				flexWrap: 'wrap',
			}}
		>
			<Button
				variant='outlined'
				color='inherit'
				onClick={() => setShowFilters(prev => !prev)}
				sx={{
					flexShrink: 0,
					height: 40,
					borderColor: '#c4c4c4',
					order: { xs: 0, sm: 0 },
					textTransform: 'none',
					color: 'text.secondary',
					whiteSpace: 'nowrap',
					fill: showFilters ? 'primary.main' : '#9ca3af',
				}}
			>
				<FilterIcon sx={{ fontSize: 20, mr: 0.5 }} />
				Фильтры
			</Button>

			<TextField
				value={filters.search}
				onChange={e => onChange({ search: e.target.value })}
				placeholder='Поиск категорий...'
				sx={{ order: { xs: 2, sm: 1 }, flex: { xs: '1 1 100%', sm: 1 }, mt: { xs: 1, sm: 0 } }}
				slotProps={{
					input: {
						startAdornment: <SearchIcon sx={{ fontSize: 16, mr: 1, fill: '#9ca3af' }} />,
					},
				}}
			/>

			<Button
				onClick={onReset}
				variant='outlined'
				color='inherit'
				sx={{
					flexShrink: 0,
					height: 40,
					order: { xs: 1, sm: 2 },
					textTransform: 'none',
					color: 'text.secondary',
					minWidth: 'auto',
					whiteSpace: 'nowrap',
					borderColor: '#c4c4c4',
				}}
			>
				<RefreshIcon sx={{ fontSize: 14, mr: 0.5, fill: '#9ca3af' }} />
				Сбросить
			</Button>

			{showFilters && (
				<Box
					sx={{
						display: 'flex',
						flexDirection: { xs: 'column', sm: 'row' },
						gap: 1.5,
						alignItems: 'end',
						width: '100%',
						order: { xs: 1, sm: 3 },
						mt: { xs: 0, sm: 1.5 },
					}}
				>
					<Box
						sx={{
							width: { xs: '100%', sm: 'auto' },
							flex: { xs: '1 1 auto', sm: 1 },
							minWidth: { xs: 'auto', sm: 180 },
						}}
					>
						<Typography
							variant='caption'
							sx={{
								fontWeight: 600,
								mb: 0.25,
								display: 'block',
								color: 'text.secondary',
								fontSize: '0.7rem',
							}}
						>
							Группа
						</Typography>
						<Select value={filters.group} onChange={e => onChange({ group: e.target.value })} fullWidth>
							<MenuItem value='all'>Все группы</MenuItem>
							{groups.map(g => (
								<MenuItem key={g.id} value={g.id}>
									{g.name}
								</MenuItem>
							))}
						</Select>
					</Box>

					<Box
						sx={{
							width: { xs: '100%', sm: 'auto' },
							flex: { xs: '1 1 auto', sm: 1 },
							minWidth: { xs: 'auto', sm: 140 },
						}}
					>
						<Typography
							variant='caption'
							sx={{
								fontWeight: 600,
								mb: 0.25,
								display: 'block',
								color: 'text.secondary',
								fontSize: '0.7rem',
							}}
						>
							Статус
						</Typography>
						<Select value={filters.status} onChange={e => onChange({ status: e.target.value })} fullWidth>
							<MenuItem value='all'>Все</MenuItem>
							<MenuItem value='active'>Активные</MenuItem>
							<MenuItem value='inactive'>Неактивные</MenuItem>
						</Select>
					</Box>

					<Box
						sx={{
							width: { xs: '100%', sm: 'auto' },
							flex: { xs: '1 1 auto', sm: 1 },
							minWidth: { xs: 'auto', sm: 140 },
						}}
					>
						<Typography
							variant='caption'
							sx={{
								fontWeight: 600,
								mb: 0.25,
								display: 'block',
								color: 'text.secondary',
								fontSize: '0.7rem',
							}}
						>
							Приоритет
						</Typography>
						<Select
							value={filters.priority}
							onChange={e => onChange({ priority: e.target.value })}
							fullWidth
						>
							<MenuItem value='all'>Все приоритеты</MenuItem>
							{Object.entries(PRIORITY_MAP).map(([value, info]) => (
								<MenuItem key={value} value={value}>
									{info.label}
								</MenuItem>
							))}
						</Select>
					</Box>
				</Box>
			)}
		</Box>
	)
}
