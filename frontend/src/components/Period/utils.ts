import dayjs from 'dayjs'
import type { DateRange, Period } from './types'

export const getDateRange = (period: Period, customRange?: DateRange): DateRange | null => {
	const end = dayjs().endOf('day')
	let start = dayjs().startOf('day')

	switch (period) {
		case 'today':
			// start уже равен текущему дню
			break
		case 'week':
			start = end.subtract(7, 'day')
			break
		case 'month':
			start = end.subtract(30, 'day')
			break
		case 'quarter':
			start = end.subtract(3, 'month')
			break
		case 'year':
			start = end.subtract(1, 'year')
			break
		case 'custom':
			if (!customRange) return null
			return {
				startDate: dayjs(customRange.startDate).toISOString(),
				endDate: dayjs(customRange.endDate).toISOString(),
			}
	}

	return {
		startDate: start.toISOString(),
		endDate: end.toISOString(),
	}
}

export const detectPeriod = (range: DateRange): Period => {
	const start = dayjs(range.startDate).startOf('day')
	const end = dayjs(range.endDate).startOf('day')
	const today = dayjs().startOf('day')

	if (start.isSame(today, 'day') && end.isSame(today, 'day')) return 'today'
	if (start.isSame(today.subtract(7, 'day'), 'day') && end.isSame(today, 'day')) return 'week'
	if (start.isSame(today.subtract(1, 'month'), 'day') && end.isSame(today, 'day')) return 'month'
	if (start.isSame(today.subtract(3, 'month'), 'day') && end.isSame(today, 'day')) return 'quarter'
	if (start.isSame(today.subtract(1, 'year'), 'day') && end.isSame(today, 'day')) return 'year'

	return 'custom'
}

export const formatDateLabel = (start: string, end: string) =>
	`${dayjs(start).format('DD.MM.YYYY')} – ${dayjs(end).format('DD.MM.YYYY')}`

export const getPresetDates = (preset: Period, type: 'day' | 'month' | 'year', count?: number): DateRange => {
	const end = dayjs().endOf('day')
	let start = dayjs().startOf('day')
	if (preset != 'today') {
		start = dayjs()
			.subtract(count || 1, type)
			.startOf('day')
	}

	return {
		startDate: start.toISOString(),
		endDate: end.toISOString(),
	}
}
