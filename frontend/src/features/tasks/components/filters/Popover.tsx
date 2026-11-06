import { useState, type FC } from 'react'
import {
	Box,
	Button,
	TextField,
	Select,
	MenuItem,
	Autocomplete,
	Popover as MuiPopover,
	Chip,
	Typography,
} from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import dayjs from 'dayjs'

import type { TicketStatus, Priority } from '../../types/task'
import type { FilterValues } from './types'
import { PermRules } from '@/features/access/constants/permissions'
import { STATUS_OPTIONS, PRIORITY_MAP } from '../../constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { getHasNoGroups } from '@/features/user/userSlice'
import { DateTextField } from '@/components/DatePicker/DatePicker'

interface Option {
	id: string
	label: string
}

interface Props {
	open: boolean
	anchorEl: HTMLElement | null
	onClose: () => void
	filters: FilterValues
	onApply: (patch: Partial<FilterValues>) => void
	siteOptions: Option[]
	ownerOptions: Option[]
	assigneeOptions: Option[]
	statusOptions: TicketStatus[]
	equipmentOptions: Option[]
}

const PRIORITY_OPTIONS = Object.entries(PRIORITY_MAP).map(([value, info]) => ({
	value: value as Priority,
	label: info.label,
}))

const SECTION_TITLE_SX = {
	fontSize: 12,
	fontWeight: 600,
	color: '#57606a',
	textTransform: 'uppercase' as const,
	mb: 0.75,
}

const SECTION_BOX_SX = {
	mb: 2,
	pb: 2,
	borderBottom: '1px solid #eaeef2',
}

export const Popover: FC<Props> = ({
	open,
	anchorEl,
	onClose,
	filters,
	onApply,
	siteOptions,
	ownerOptions,
	assigneeOptions,
	statusOptions,
	equipmentOptions,
}) => {
	const hasNoGroups = useAppSelector(getHasNoGroups)
	const canSeeEquipment = useCheckPermissions({ allOf: [PermRules.Equipment.Read] })

	// Инициализируем стейт один раз при монтировании (благодаря key в родительском компоненте)
	const [local, setLocal] = useState<FilterValues>(filters)

	const update = <K extends keyof FilterValues>(key: K, value: FilterValues[K]) => {
		setLocal(prev => ({ ...prev, [key]: value }))
	}

	const handleApply = () => {
		onApply({
			ticketNumber: local.ticketNumber || undefined,
			ownerId: local.ownerId || undefined,
			siteIds: local.siteIds?.length ? local.siteIds : undefined,
			dueDateFrom: hasNoGroups ? undefined : local.dueDateFrom || undefined,
			dueDateTo: hasNoGroups ? undefined : local.dueDateTo || undefined,
			priorities: hasNoGroups || !local.priorities?.length ? undefined : local.priorities,
			assigneeId: local.assigneeId || undefined,
			statuses: local.statuses?.length ? local.statuses : undefined,
			equipmentId: local.equipmentId || undefined,
		})
		onClose()
	}

	const handleReset = () => {
		setLocal(prev => ({
			...prev,
			ticketNumber: '',
			ownerId: undefined,
			siteIds: [],
			dueDateFrom: '',
			dueDateTo: '',
			priorities: [],
			assigneeId: undefined,
			statuses: [],
			equipmentId: undefined,
		}))
	}

	// Ленивый рендеринг: если поповер закрыт, не грузим DOM
	if (!open) return null

	const currentOwner = ownerOptions.find(u => u.id === local.ownerId) ?? null
	const currentSites = siteOptions.filter(s => local.siteIds?.includes(s.id))
	const currentAssignee = assigneeOptions.find(u => u.id === local.assigneeId) ?? null
	const currentEquipment = equipmentOptions.find(e => e.id === local.equipmentId) ?? null

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
				{/* № заявки */}
				<Box sx={SECTION_BOX_SX}>
					<Typography sx={SECTION_TITLE_SX}>№ заявки</Typography>
					<TextField
						type='number'
						fullWidth
						size='small'
						placeholder='Например: 1024'
						value={local.ticketNumber ?? ''}
						onChange={e => update('ticketNumber', e.target.value)}
					/>
				</Box>

				{/* Заказчик */}
				<Box sx={SECTION_BOX_SX}>
					<Typography sx={SECTION_TITLE_SX}>Заказчик</Typography>
					<Autocomplete
						options={ownerOptions}
						value={currentOwner}
						onChange={(_, v) => update('ownerId', v?.id)}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} size='small' placeholder='Выберите...' />}
						noOptionsText='Нет пользователей'
					/>
				</Box>

				{/* Площадка */}
				<Box sx={SECTION_BOX_SX}>
					<Typography sx={SECTION_TITLE_SX}>Площадка</Typography>
					<Autocomplete
						multiple
						disableCloseOnSelect
						options={siteOptions}
						value={currentSites}
						onChange={(_, v) =>
							update(
								'siteIds',
								v.map(s => s.id),
							)
						}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} size='small' placeholder='Выберите...' />}
						noOptionsText='Нет площадок'
					/>
				</Box>

				{/* Оборудование */}
				{canSeeEquipment && (
					<Box sx={SECTION_BOX_SX}>
						<Typography sx={SECTION_TITLE_SX}>Оборудование</Typography>
						<Autocomplete
							options={equipmentOptions}
							value={currentEquipment}
							onChange={(_, v) => update('equipmentId', v?.id)}
							getOptionLabel={o => o.label}
							renderInput={params => <TextField {...params} size='small' placeholder='Выберите...' />}
							noOptionsText='Нет оборудования'
						/>
					</Box>
				)}

				{/* Срок */}
				{!hasNoGroups && (
					<Box sx={SECTION_BOX_SX}>
						<Typography sx={SECTION_TITLE_SX}>Срок</Typography>
						<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 1 }}>
							<DatePicker
								label='С'
								value={local.dueDateFrom ? dayjs(local.dueDateFrom) : null}
								onChange={date => update('dueDateFrom', date ? date.format('YYYY-MM-DD') : '')}
								slots={{ textField: DateTextField }}
								slotProps={{ textField: { fullWidth: true, size: 'small' } }}
							/>
							<DatePicker
								label='По'
								value={local.dueDateTo ? dayjs(local.dueDateTo) : null}
								onChange={date => update('dueDateTo', date ? date.format('YYYY-MM-DD') : '')}
								slots={{ textField: DateTextField }}
								slotProps={{ textField: { fullWidth: true, size: 'small' } }}
							/>
						</Box>
					</Box>
				)}

				{/* Приоритет */}
				{!hasNoGroups && (
					<Box sx={SECTION_BOX_SX}>
						<Typography sx={SECTION_TITLE_SX}>Приоритет</Typography>
						<Select
							multiple
							fullWidth
							size='small'
							value={local.priorities ?? []}
							onChange={e => update('priorities', e.target.value as Priority[])}
							renderValue={selected => (
								<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
									{selected.map(v => (
										<Chip
											key={v}
											label={PRIORITY_OPTIONS.find(o => o.value === v)?.label ?? v}
											size='small'
										/>
									))}
								</Box>
							)}
						>
							{PRIORITY_OPTIONS.map(o => (
								<MenuItem key={o.value} value={o.value}>
									{o.label}
								</MenuItem>
							))}
						</Select>
					</Box>
				)}

				{/* Исполнитель */}
				<Box sx={SECTION_BOX_SX}>
					<Typography sx={SECTION_TITLE_SX}>Исполнитель</Typography>
					<Autocomplete
						options={assigneeOptions}
						value={currentAssignee}
						onChange={(_, v) => update('assigneeId', v?.id)}
						getOptionLabel={o => o.label}
						renderInput={params => <TextField {...params} size='small' placeholder='Выберите...' />}
						noOptionsText='Нет пользователей'
					/>
				</Box>

				{/* Статус */}
				<Box sx={{ mb: 1 }}>
					<Typography sx={SECTION_TITLE_SX}>Статус</Typography>
					<Select
						multiple
						fullWidth
						size='small'
						value={local.statuses ?? []}
						onChange={e => update('statuses', e.target.value as TicketStatus[])}
						renderValue={selected => (
							<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
								{selected.map(v => (
									<Chip
										key={v}
										label={STATUS_OPTIONS.find(o => o.value === v)?.label ?? v}
										size='small'
									/>
								))}
							</Box>
						)}
					>
						{statusOptions.map(o => (
							<MenuItem key={o} value={o}>
								{STATUS_OPTIONS.find(s => s.value === o)?.label ?? o}
							</MenuItem>
						))}
					</Select>
				</Box>

				{/* Кнопки действий */}
				<Box
					sx={{
						display: 'flex',
						justifyContent: 'space-between',
						gap: 1,
						mt: 2,
						pt: 2,
						borderTop: '1px solid #eaeef2',
					}}
				>
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
					<Button
						variant='contained'
						onClick={handleApply}
						sx={{ textTransform: 'none', borderRadius: '6px' }}
					>
						Применить
					</Button>
				</Box>
			</Box>
		</MuiPopover>
	)
}
