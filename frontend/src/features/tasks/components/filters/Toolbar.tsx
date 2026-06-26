import { useState, type FC } from 'react'
import { Box, Button, TextField, Menu, MenuItem } from '@mui/material'

import type { GroupByField } from '../../constants/taskMaps'
import { GROUP_BY_OPTIONS } from '../../constants/taskMaps'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { FilterIcon } from '@/components/Icons/FilterIcon'
import { StackIcon } from '@/components/Icons/StackIcon'
import { RefreshIcon } from '@/components/Icons/RefreshIcon'

interface Props {
	search: string
	groupEnabled: boolean
	groupBy: string
	activeCount: number
	onSearchChange: (value: string) => void
	onGroupChange: (option?: GroupByField) => void
	onOpenFilter: (e: React.MouseEvent<HTMLElement>) => void
	onReset: () => void
}

export const Toolbar: FC<Props> = ({
	search,
	groupEnabled,
	groupBy,
	activeCount,
	onSearchChange,
	onGroupChange,
	onOpenFilter,
	onReset,
}) => {
	const [groupAnchorEl, setGroupAnchorEl] = useState<HTMLElement | null>(null)

	return (
		<Box
			sx={{
				bgcolor: '#fff',
				p: 2.5,
				borderRadius: '12px',
				border: '1px solid #e5e7eb',
				mb: 2,
				display: 'flex',
				alignItems: 'center',
				gap: 1,
				flexWrap: 'wrap',
			}}
		>
			<TextField
				value={search}
				onChange={e => onSearchChange(e.target.value)}
				placeholder='Поиск...'
				sx={{ flex: { xs: '1 1 100%', sm: 1 }, order: { xs: 0, sm: 0 }, minWidth: { xs: 150, sm: 250 } }}
				slotProps={{
					input: {
						startAdornment: <SearchIcon sx={{ fontSize: 16, mr: 1, fill: '#9ca3af' }} />,
					},
				}}
			/>

			<Box sx={{ width: { xs: '100%', sm: 'auto' } }}>
				<Button
					variant='outlined'
					color='inherit'
					onClick={e => setGroupAnchorEl(e.currentTarget)}
					fullWidth
					sx={{
						flexShrink: 0,
						height: 40,
						borderColor: '#c4c4c4',
						textTransform: 'none',
						color: 'text.secondary',
						whiteSpace: 'nowrap',
						fill: groupEnabled ? 'primary.main' : '#9ca3af',
					}}
				>
					<StackIcon sx={{ fontSize: 18, mr: 1 }} />
					Группировка
				</Button>
				<Menu
					open={Boolean(groupAnchorEl)}
					anchorEl={groupAnchorEl}
					onClose={() => setGroupAnchorEl(null)}
				>
					<MenuItem value='none' selected={!groupEnabled} onClick={() => { onGroupChange(); setGroupAnchorEl(null) }}>
						Без группировки
					</MenuItem>
					{GROUP_BY_OPTIONS.map(opt => (
						<MenuItem
							key={opt.value}
							value={opt.value}
							selected={opt.value === groupBy && groupEnabled}
							onClick={() => { onGroupChange(opt.value); setGroupAnchorEl(null) }}
						>
							{opt.label}
						</MenuItem>
					))}
				</Menu>
			</Box>

			<Button
				variant='outlined'
				color='inherit'
				onClick={onOpenFilter}
				sx={{
					flexShrink: 0,
					height: 40,
					borderColor: '#c4c4c4',
					textTransform: 'none',
					color: 'text.secondary',
					whiteSpace: 'nowrap',
					fill: activeCount > 0 ? 'primary.main' : '#9ca3af',
					width: { xs: '100%', sm: 'auto' },
				}}
			>
				<FilterIcon sx={{ fontSize: 18, mr: 1 }} />
				Фильтры
				{activeCount > 0 && (
					<Box
						component='span'
						sx={{
							ml: 0.75,
							bgcolor: '#2f81f7',
							color: '#fff',
							borderRadius: '10px',
							px: 0.75,
							py: 0.125,
							fontSize: '0.75rem',
							fontWeight: 600,
							lineHeight: 1.4,
						}}
					>
						{activeCount}
					</Box>
				)}
			</Button>

			<Button
				onClick={onReset}
				variant='outlined'
				color='inherit'
				sx={{
					flexShrink: 0,
					height: 40,
					order: { xs: 1, sm: 3 },
					textTransform: 'none',
					color: 'text.secondary',
					minWidth: 'auto',
					whiteSpace: 'nowrap',
					borderColor: '#c4c4c4',
					width: { xs: '100%', sm: 'auto' },
				}}
			>
				<RefreshIcon sx={{ fontSize: 14, mr: 1, fill: '#9ca3af' }} />
				Сбросить
			</Button>
		</Box>
	)
}
