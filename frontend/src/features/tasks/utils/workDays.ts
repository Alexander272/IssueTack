import dayjs, { type Dayjs } from 'dayjs'
import type { IProductionCalendar } from '@/features/calendar/types'

type ExceptionMap = Map<string, boolean>

const toExceptionMap = (calendar?: IProductionCalendar | null): ExceptionMap => {
	const map: ExceptionMap = new Map()
	for (const ex of calendar?.exceptions ?? []) {
		map.set(ex.date, ex.isWorking)
	}
	return map
}

const isWorkingDay = (d: Dayjs, exceptions: ExceptionMap): boolean => {
	const key = d.format('YYYY-MM-DD')
	const ex = exceptions.get(key)
	if (ex !== undefined) return ex
	const wd = d.day() // 0=Sun ... 6=Sat
	return wd !== 0 && wd !== 6
}

const resolveTimezone = (calendar?: IProductionCalendar | null): string => {
	const fallback = dayjs.tz.guess()
	const tz = calendar?.timezone?.trim()
	if (!tz || tz.toLowerCase() === 'local') return fallback
	try {
		// dayjs.tz бросает RangeError на невалидной IANA-зоне (например, "Local")
		dayjs().tz(tz)
		return tz
	} catch {
		return fallback
	}
}

/**
 * Возвращает конец N-го рабочего дня после base в календарной таймзоне.
 * База — точка отсчёта: первый засчитанный день — следующий рабочий
 * (даже если base сам рабочий). При отсутствии данных календаря используется
 * фолбэк Mon–Fri в локальной таймзоне.
 */
export const addWorkingDays = (
	base: Dayjs | string | Date,
	days: number,
	calendar?: IProductionCalendar | null,
): Dayjs => {
	if (days <= 0) return dayjs(base)

	const tz = resolveTimezone(calendar)
	const exceptions = toExceptionMap(calendar)

	let cur = dayjs(base).tz(tz).startOf('day')
	let added = 0
	while (added < days) {
		cur = cur.add(1, 'day')
		if (isWorkingDay(cur, exceptions)) added++
	}
	return cur.endOf('day')
}

/**
 * Удобный хелпер для форм: считает автосрок в рабочих днях от now/base и
 * возвращает ISO-строку (или null при отсутствии дней).
 */
export const autoDueDateISO = (
	days: number | null | undefined,
	calendar?: IProductionCalendar | null,
	base?: Dayjs | string | Date | null,
): string | null => {
	if (days == null) return null
	const b = base ? dayjs(base) : dayjs()
	return addWorkingDays(b, days, calendar).toISOString()
}
