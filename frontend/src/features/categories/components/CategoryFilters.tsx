import { type FC } from 'react'
import { Box, Button, MenuItem, Select, TextField, Typography } from '@mui/material'

import type { IGroup } from '@/features/groups/types/group'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { SearchIcon } from '@/components/Icons/SearchIcon'
import { RefreshIcon } from '@/components/Icons/RefreshIcon'

type Props = {
	groups: IGroup[]
	filterGroup: string
	filterStatus: string
	filterPriority: string
	search: string
	onGroupChange: (v: string) => void
	onStatusChange: (v: string) => void
	onPriorityChange: (v: string) => void
	onSearchChange: (v: string) => void
	onReset: () => void
}

export const CategoryFilters: FC<Props> = ({
	groups,
	filterGroup,
	filterStatus,
	filterPriority,
	search,
	onGroupChange,
	onStatusChange,
	onPriorityChange,
	onSearchChange,
	onReset,
}) => {
	return (
		<Box
			sx={{
				bgcolor: '#fff',
				p: 2,
				borderRadius: '12px',
				border: '1px solid #e5e7eb',
				mb: 3,
				display: 'flex',
				flexDirection: 'column',
				gap: 2,
			}}
		>
			<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'end' }}>
				<Box sx={{ flex: '1 1 200px', minWidth: 200 }}>
					<Typography
						variant='caption'
						sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: 'text.secondary' }}
					>
						Группа-владелец
					</Typography>
					<Select
						value={filterGroup}
						onChange={e => onGroupChange(e.target.value)}
						fullWidth
						size='small'
						sx={{ borderRadius: '8px' }}
					>
						<MenuItem value='all'>Все группы</MenuItem>
						{groups.map(g => (
							<MenuItem key={g.id} value={g.id}>
								{g.name}
							</MenuItem>
						))}
					</Select>
				</Box>

				<Box sx={{ flex: '1 1 200px', minWidth: 200 }}>
					<Typography
						variant='caption'
						sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: 'text.secondary' }}
					>
						Статус
					</Typography>
					<Select
						value={filterStatus}
						onChange={e => onStatusChange(e.target.value)}
						fullWidth
						size='small'
						sx={{ borderRadius: '8px' }}
					>
						<MenuItem value='all'>Все</MenuItem>
						<MenuItem value='active'>Только активные</MenuItem>
						<MenuItem value='inactive'>Только неактивные</MenuItem>
					</Select>
				</Box>

				<Box sx={{ flex: '1 1 200px', minWidth: 200 }}>
					<Typography
						variant='caption'
						sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: 'text.secondary' }}
					>
						Приоритет
					</Typography>
					<Select
						value={filterPriority}
						onChange={e => onPriorityChange(e.target.value)}
						fullWidth
						size='small'
						sx={{ borderRadius: '8px' }}
					>
						<MenuItem value='all'>Все приоритеты</MenuItem>
						{Object.entries(PRIORITY_MAP).map(([value, info]) => (
							<MenuItem key={value} value={value}>
								{info.label}
							</MenuItem>
						))}
					</Select>
				</Box>

				<Button
					onClick={onReset}
					variant='text'
					sx={{ textTransform: 'none', color: 'text.secondary', minWidth: 'auto' }}
				>
					<RefreshIcon sx={{ fontSize: 16, mr: 1, fill: '#9ca3af' }} />
					Сбросить
				</Button>
			</Box>

			<Box sx={{ display: 'flex', justifyContent: 'center' }}>
				<TextField
					value={search}
					onChange={e => onSearchChange(e.target.value)}
					placeholder='Поиск категорий...'
					fullWidth
					slotProps={{
						input: {
							startAdornment: <SearchIcon sx={{ fontSize: 16, mr: 1, fill: '#9ca3af' }} />,
						},
					}}
					sx={{ maxWidth: 900 }}
				/>
			</Box>
		</Box>
	)
}
