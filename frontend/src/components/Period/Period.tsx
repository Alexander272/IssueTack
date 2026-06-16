import { useState, useMemo, type FC } from 'react'
import { Box, Popover, Stack, Divider, Typography } from '@mui/material'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import dayjs, { Dayjs } from 'dayjs'

import { CalendarIcon } from '../Icons/CalendarIcon'
import { DateTextField } from '../DatePicker/DatePicker'
import { ActionButton } from './Button'
import { PresetChip } from './PresetChip'
import { detectPeriod, formatDateLabel, getPresetDates } from './utils'
import type { DateRange, Period, Preset } from './types'

const PRESETS: Preset[] = [
	{ label: 'Сегодня', value: 'today' },
	{ label: 'Последние 7 дней', value: 'week', type: 'day', count: 7 },
	{ label: 'Последний месяц', value: 'month', type: 'month', count: 1 },
	{ label: 'Квартал', value: 'quarter', type: 'month', count: 3 },
	{ label: 'Год', value: 'year', type: 'year', count: 1 },
]

type Props = {
	value: DateRange | undefined
	onChange: (range: DateRange) => void
}

export const PeriodPicker: FC<Props> = ({ value, onChange }) => {
	const [anchorEl, setAnchorEl] = useState<HTMLDivElement | null>(null)
	const [draftStart, setDraftStart] = useState<Dayjs | null>(null)
	const [draftEnd, setDraftEnd] = useState<Dayjs | null>(null)
	const [draftPeriod, setDraftPeriod] = useState<Period>(() => (value ? detectPeriod(value) : 'week'))

	const period = useMemo(() => (value ? detectPeriod(value) : 'week'), [value])

	const open = Boolean(anchorEl)

	const handleOpen = (e: React.MouseEvent<HTMLDivElement>) => {
		setAnchorEl(e.currentTarget)

		if (value) {
			setDraftStart(dayjs(value.startDate))
			setDraftEnd(dayjs(value.endDate))
			setDraftPeriod(detectPeriod(value))
		} else {
			const preset = PRESETS.find(p => p.value === period)
			if (preset?.type && preset.count) {
				const dates = getPresetDates(preset.value, preset.type, preset.count)
				setDraftStart(dayjs(dates.startDate))
				setDraftEnd(dayjs(dates.endDate))
			} else {
				setDraftStart(null)
				setDraftEnd(null)
			}
			setDraftPeriod(period)
		}
	}

	const handleClose = () => setAnchorEl(null)

	const applyPreset = (preset: Period, type: 'day' | 'month' | 'year', count: number) => {
		const range = getPresetDates(preset, type, count)
		onChange(range)
		handleClose()
	}

	const applyCustom = () => {
		if (!draftStart || !draftEnd) return

		const range = {
			startDate: draftStart.startOf('day').toISOString(),
			endDate: draftEnd.endOf('day').toISOString(),
		}
		onChange(range)
		handleClose()
	}

	const reset = () => {
		setDraftStart(null)
		setDraftEnd(null)
		setDraftPeriod('week')
	}

	const getLabel = () => {
		if (!value) return 'Выбрать период'
		const p = detectPeriod(value)
		const preset = PRESETS.find(pr => pr.value === p)
		if (preset) return preset.label
		return formatDateLabel(value.startDate, value.endDate)
	}

	return (
		<Box>
			<Box
				onClick={handleOpen}
				sx={{
					display: 'flex',
					alignItems: 'center',
					gap: 1,
					px: 1.5,
					py: 1,
					bgcolor: 'background.paper',
					border: '1px solid',
					borderColor: 'divider',
					borderRadius: 9999,
					cursor: 'pointer',
					fontSize: '14px',
					color: 'text.primary',
					transition: '0.15s',
					'&:hover': {
						borderColor: 'primary.main',
						boxShadow: '0 2px 8px rgba(0,0,0,0.06)',
					},
					userSelect: 'none',
				}}
			>
				<CalendarIcon fontSize={18} /> <Typography sx={{ fontSize: '0.9rem' }}>{getLabel()}</Typography>
			</Box>

			<Popover
				open={open}
				anchorEl={anchorEl}
				onClose={handleClose}
				anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
				transformOrigin={{ vertical: 'top', horizontal: 'left' }}
				disableScrollLock
				slotProps={{
					paper: {
						sx: {
							width: 520,
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
					<Stack direction='row' sx={{ mb: 1, gap: 1, justifyContent: 'center', flexWrap: 'wrap' }}>
						{PRESETS.map(p => (
							<PresetChip
								key={p.value}
								preset={p}
								isActive={draftPeriod === p.value}
								onClick={() => applyPreset(p.value, p.type || 'day', p.count || 7)}
							/>
						))}
					</Stack>

					<Divider sx={{ mb: 2 }} />

					<Stack direction='row' spacing={1.5} sx={{ mb: 2 }}>
						<DatePicker
							label='С'
							value={draftStart}
							onChange={val => {
								setDraftStart(val)
								setDraftPeriod('custom')
							}}
							slots={{ textField: DateTextField }}
							slotProps={{ textField: { fullWidth: true } }}
						/>
						<DatePicker
							label='По'
							value={draftEnd}
							onChange={val => {
								setDraftEnd(val)
								setDraftPeriod('custom')
							}}
							slots={{ textField: DateTextField }}
							slotProps={{ textField: { fullWidth: true } }}
						/>
					</Stack>

					<Stack direction='row' spacing={1}>
						<Box
							onClick={reset}
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
							Сброс
						</Box>
						<ActionButton onClick={applyCustom} disabled={!draftStart || !draftEnd}>
							Применить
						</ActionButton>
					</Stack>
				</Box>
			</Popover>
		</Box>
	)
}
