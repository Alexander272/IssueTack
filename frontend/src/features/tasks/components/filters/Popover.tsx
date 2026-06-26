import { useState, type FC } from 'react'
import { Box, Button, TextField, Select, MenuItem, Autocomplete, Popover as MuiPopover, Chip, Typography } from '@mui/material'

import type { TicketStatus, Priority } from '../../types/task'
import { STATUS_OPTIONS, PRIORITY_MAP } from '../../constants/taskMaps'
import type { FilterValues } from './types'

interface Option {
	id: string
	label: string
}

interface Props {
	open: boolean
	anchorEl: HTMLElement | null
	onClose: () => void
	initial: FilterValues
	onApply: (patch: Partial<FilterValues>) => void
	siteOptions: Option[]
	userOptions: Option[]
}

const PRIORITY_OPTIONS: { value: Priority; label: string }[] = Object.entries(PRIORITY_MAP).map(([value, info]) => ({
	value: value as Priority,
	label: info.label,
}))

export const Popover: FC<Props> = ({ open, anchorEl, onClose, initial, onApply, siteOptions, userOptions }) => {
	const [local, setLocal] = useState(() => ({
		ticketNumber: initial.ticketNumber ?? '',
		ownerId: initial.ownerId ?? null as string | null,
		siteIds: initial.siteIds ?? [] as string[],
		dueDateFrom: initial.dueDateFrom ?? '',
		dueDateTo: initial.dueDateTo ?? '',
		priorities: initial.priorities ?? [] as Priority[],
		assigneeId: initial.assigneeId ?? null as string | null,
		statuses: initial.statuses ?? [] as TicketStatus[],
	}))

	const update = <K extends keyof typeof local>(key: K, value: (typeof local)[K]) => {
		setLocal(prev => ({ ...prev, [key]: value }))
	}

	const handleApply = () => {
		onApply({
			ticketNumber: local.ticketNumber || undefined,
			ownerId: local.ownerId ?? undefined,
			siteIds: local.siteIds.length > 0 ? local.siteIds : undefined,
			dueDateFrom: local.dueDateFrom || undefined,
			dueDateTo: local.dueDateTo || undefined,
			priorities: local.priorities.length > 0 ? local.priorities : undefined,
			assigneeId: local.assigneeId ?? undefined,
			statuses: local.statuses.length > 0 ? local.statuses : undefined,
		})
		onClose()
	}

	const handleReset = () => {
		setLocal({
			ticketNumber: '',
			ownerId: null,
			siteIds: [],
			dueDateFrom: '',
			dueDateTo: '',
			priorities: [],
			assigneeId: null,
			statuses: [],
		})
	}

	const currentOwner = userOptions.find(u => u.id === local.ownerId) ?? null
	const currentSites = siteOptions.filter(s => local.siteIds.includes(s.id))
	const currentAssignee = userOptions.find(u => u.id === local.assigneeId) ?? null

	const sectionSx = { fontSize: 12, fontWeight: 600, color: '#57606a', textTransform: 'uppercase' as const, mb: 0.75 }

	return (
		<MuiPopover
			open={open}
			anchorEl={anchorEl}
			onClose={onClose}
			anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
			transformOrigin={{ vertical: 'top', horizontal: 'right' }}
			slotProps={{
				paper: {
					sx: { borderRadius: '10px', boxShadow: '0 10px 30px rgba(0,0,0,0.12)', mt: 0.5 },
				},
			}}
		>
			<Box sx={{ width: 380, p: 2 }}>
				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>№ заявки</Typography>
					<TextField
						type='number'
						fullWidth
						placeholder='Например: 1024'
						value={local.ticketNumber}
						onChange={e => update('ticketNumber', e.target.value)}
					/>
				</Box>

				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>Заказчик</Typography>
					<Autocomplete
						options={userOptions}
						value={currentOwner}
						onChange={(_, v) => update('ownerId', v?.id ?? null)}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} placeholder='Выберите...' />}
						noOptionsText='Нет пользователей'
					/>
				</Box>

				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>Площадка</Typography>
					<Autocomplete
						multiple
						options={siteOptions}
						value={currentSites}
						onChange={(_, v) => update('siteIds', v.map(s => s.id))}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} placeholder='Выберите...' />}
						noOptionsText='Нет площадок'
					/>
				</Box>

				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>Срок</Typography>
					<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1 }}>
						<TextField
							type='date'
							label='С'
							value={local.dueDateFrom}
							onChange={e => update('dueDateFrom', e.target.value)}
							slotProps={{ inputLabel: { shrink: true } }}
						/>
						<TextField
							type='date'
							label='По'
							value={local.dueDateTo}
							onChange={e => update('dueDateTo', e.target.value)}
							slotProps={{ inputLabel: { shrink: true } }}
						/>
					</Box>
				</Box>

				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>Приоритет</Typography>
					<Select
						multiple
						fullWidth
						value={local.priorities}
						onChange={e => update('priorities', e.target.value as Priority[])}
						renderValue={selected => (
							<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
								{selected.map(v => (
									<Chip key={v} label={PRIORITY_OPTIONS.find(o => o.value === v)?.label ?? v} size='small' />
								))}
							</Box>
						)}
					>
						{PRIORITY_OPTIONS.map(o => (
							<MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
						))}
					</Select>
				</Box>

				<Box sx={{ mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }}>
					<Typography sx={sectionSx}>Назначено</Typography>
					<Autocomplete
						options={userOptions}
						value={currentAssignee}
						onChange={(_, v) => update('assigneeId', v?.id ?? null)}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} placeholder='Выберите...' />}
						noOptionsText='Нет пользователей'
					/>
				</Box>

				<Box sx={{ mb: 2 }}>
					<Typography sx={sectionSx}>Статус</Typography>
					<Select
						multiple
						fullWidth
						value={local.statuses}
						onChange={e => update('statuses', e.target.value as TicketStatus[])}
						renderValue={selected => (
							<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
								{selected.map(v => (
									<Chip key={v} label={STATUS_OPTIONS.find(o => o.value === v)?.label ?? v} size='small' />
								))}
							</Box>
						)}
					>
						{STATUS_OPTIONS.filter(o => o.value !== 'all').map(o => (
							<MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
						))}
					</Select>
				</Box>

				<Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, mt: 2, pt: 2, borderTop: '1px solid #eaeef2' }}>
					<Button
						onClick={handleReset}
						sx={{
							color: '#2f81f7',
							textTransform: 'none',
							p: 0,
							minWidth: 'auto',
							'&:hover': { textDecoration: 'underline', bgcolor: 'transparent' },
						}}
					>
						Сбросить всё
					</Button>
					<Button variant='contained' onClick={handleApply} sx={{ textTransform: 'none', borderRadius: '6px' }}>
						Применить
					</Button>
				</Box>
			</Box>
		</MuiPopover>
	)
}
