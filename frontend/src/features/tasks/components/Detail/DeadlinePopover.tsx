import { useState, type FC, type ComponentType } from 'react'
import { Box, Button, Popover, Divider, Stack, Typography, Chip, type SvgIconProps } from '@mui/material'
import { CalendarClock, Zap, Hourglass, Clock, CalendarDays, CalendarRange } from 'lucide-mui'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import dayjs, { Dayjs } from 'dayjs'

import { DateTextField } from '@/components/DatePicker/DatePicker'
import { ActionButton } from '@/components/Period/Button'

type Props = {
	onSetDueDate: (iso: string) => void
}

type Preset = {
	label: string
	getDate: () => Dayjs
	Icon: ComponentType<SvgIconProps>
	bg: string
	color: string
}

const PRESETS: Preset[] = [
	{
		label: 'Сегодня',
		getDate: () => dayjs().endOf('day'),
		Icon: Zap,
		bg: 'rgba(239,68,68,0.12)',
		color: '#dc2626',
	},
	{
		label: 'Завтра',
		getDate: () => dayjs().add(1, 'day').endOf('day'),
		Icon: Hourglass,
		bg: 'rgba(249,115,22,0.12)',
		color: '#ea580c',
	},
	{
		label: 'Через 3 дня',
		getDate: () => dayjs().add(3, 'day').endOf('day'),
		Icon: Clock,
		bg: 'rgba(245,158,11,0.14)',
		color: '#d97706',
	},
	{
		label: 'Через неделю',
		getDate: () => dayjs().add(7, 'day').endOf('day'),
		Icon: CalendarDays,
		bg: 'rgba(59,130,246,0.12)',
		color: '#3b82f6',
	},
	{
		label: 'Конец недели',
		getDate: () => dayjs().endOf('week').endOf('day'),
		Icon: CalendarRange,
		bg: 'rgba(99,102,241,0.12)',
		color: '#6366f1',
	},
	{
		label: 'Конец месяца',
		getDate: () => dayjs().endOf('month').endOf('day'),
		Icon: CalendarClock,
		bg: 'rgba(147,51,234,0.12)',
		color: '#7c3aed',
	},
]

const PresetChip: FC<{ preset: Preset; onClick: () => void }> = ({ preset, onClick }) => {
	const { label, Icon, bg, color } = preset

	return (
		<Chip
			icon={<Icon sx={{ fontSize: 16 }} />}
			label={label}
			onClick={onClick}
			sx={{
				bgcolor: bg,
				color,
				fontWeight: 600,
				cursor: 'pointer',
				pl: 0.5,
				'& .MuiChip-icon': { color },
				transition: '0.15s',
				'&:hover': { boxShadow: '0 2px 8px rgba(0,0,0,0.12)' },
			}}
		/>
	)
}

export const DeadlinePopover: FC<Props> = ({ onSetDueDate }) => {
	const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null)
	const [customDate, setCustomDate] = useState<Dayjs | null>(null)

	const open = Boolean(anchorEl)

	const close = () => {
		setAnchorEl(null)
		setCustomDate(null)
	}

	const apply = (date: Dayjs | null) => {
		if (!date) return
		onSetDueDate(date.endOf('day').toISOString())
		close()
	}

	return (
		<>
			<Button
				variant='outlined'
				onClick={e => setAnchorEl(e.currentTarget)}
				startIcon={<CalendarClock sx={{ fontSize: 16 }} />}
				sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
			>
				Установить срок
			</Button>

			<Popover
				open={open}
				anchorEl={anchorEl}
				onClose={close}
				anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
				transformOrigin={{ vertical: 'top', horizontal: 'right' }}
				disableScrollLock
				slotProps={{
					paper: {
						sx: {
							width: 340,
							borderRadius: 3,
							border: '1px solid',
							borderColor: 'divider',
							boxShadow: '0 12px 32px rgba(0,0,0,0.1)',
							mt: 1,
						},
					},
				}}
			>
				<Box sx={{ p: 2 }}>
					<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mb: 2 }}>
						{PRESETS.map(p => (
							<PresetChip key={p.label} preset={p} onClick={() => apply(p.getDate())} />
						))}
					</Box>

					<Divider sx={{ mb: 2 }} />

					<Stack spacing={0.5} sx={{ mb: 2 }}>
						<Typography variant='caption' sx={{ color: 'text.secondary' }}>
							Другая дата
						</Typography>
						<DatePicker
							value={customDate}
							onChange={setCustomDate}
							slots={{ textField: DateTextField }}
							slotProps={{ textField: { fullWidth: true, size: 'small' } }}
							disablePast
						/>
					</Stack>

					<Stack direction='row' spacing={1}>
						<Box
							onClick={close}
							sx={{
								flex: 1,
								textAlign: 'center',
								py: 1,
								borderRadius: 2,
								bgcolor: 'grey.100',
								cursor: 'pointer',
								fontSize: '14px',
								userSelect: 'none',
								'&:hover': { bgcolor: 'grey.200' },
							}}
						>
							Отмена
						</Box>
						<ActionButton onClick={() => apply(customDate)} disabled={!customDate}>
							Применить
						</ActionButton>
					</Stack>
				</Box>
			</Popover>
		</>
	)
}
