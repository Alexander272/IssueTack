// Расчёт автосрока в рабочих днях для плагина MM. Логика зеркалит бэкенд
// (services/work_calendar.go) и веб (features/tasks/utils/workDays.ts):
// N рабочих дней от базы (база — точка отсчёта, не засчитывается), конец
// целевого рабочего дня в календарной таймзоне. Исключения — по civil-дате
// (YYYY-MM-DD) в календарной TZ.

import type { PluginCalendar } from './types'

type ExceptionMap = Map<string, boolean>

const exceptionMap = (calendar?: PluginCalendar | null): ExceptionMap => {
	const map: ExceptionMap = new Map()
	for (const ex of calendar?.exceptions || []) {
		map.set(ex.date, ex.isWorking)
	}
	return map
}

const pad2 = (n: number) => String(n).padStart(2, '0')

const dateKey = (y: number, m: number, d: number) => `${y}-${pad2(m)}-${pad2(d)}`

const isValidTimezone = (tz: string): boolean => {
	try {
		Intl.DateTimeFormat(undefined, { timeZone: tz })
		return true
	} catch {
		return false
	}
}

const resolveTimezone = (calendar?: PluginCalendar | null): string => {
	const fallback = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
	const tz = calendar?.timezone?.trim()
	if (!tz || tz.toLowerCase() === 'local') return fallback
	return isValidTimezone(tz) ? tz : fallback
}

// civilDateInTZ возвращает компоненты civil-даты момента в IANA-таймзоне.
const civilDateInTZ = (date: Date, tz: string): { y: number; m: number; d: number } => {
	const parts = new Intl.DateTimeFormat('en-US', {
		timeZone: tz,
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
	}).formatToParts(date)
	const get = (type: string) => Number(parts.find(p => p.type === type)?.value)
	return { y: get('year'), m: get('month'), d: get('day') }
}

const isWorkingDay = (y: number, m: number, d: number, exceptions: ExceptionMap): boolean => {
	const ex = exceptions.get(dateKey(y, m, d))
	if (ex !== undefined) return ex
	// Дефолт: Пн–Пт рабочие, Сб/Вс выходные. Weekday civil-даты не зависит от TZ.
	const wd = new Date(Date.UTC(y, m - 1, d)).getUTCDay() // 0=Sun..6=Sat
	return wd !== 0 && wd !== 6
}

// addWorkingDaysEnd возвращает civil-дату (конец дня) N-го рабочего дня после base
// в календарной TZ: { y,m,d } целевого дня. При days <= 0 — civil-дата base.
export const addWorkingDaysEnd = (
	base: Date,
	days: number,
	calendar?: PluginCalendar | null,
): { y: number; m: number; d: number } => {
	const tz = resolveTimezone(calendar)
	const ex = exceptionMap(calendar)
	const start = civilDateInTZ(base, tz)
	let cursor = Date.UTC(start.y, start.m - 1, start.d)
	let y = start.y
	let m = start.m
	let d = start.d
	if (days <= 0) {
		return { y, m, d }
	}
	let added = 0
	while (added < days) {
		cursor += 86400000
		const dt = new Date(cursor)
		y = dt.getUTCFullYear()
		m = dt.getUTCMonth() + 1
		d = dt.getUTCDate()
		if (isWorkingDay(y, m, d, ex)) added++
	}
	return { y, m, d }
}

// toDatetimeLocal форматирует civil-дату как конец дня для <input type="datetime-local">.
export const toDatetimeLocal = (c: { y: number; m: number; d: number }): string =>
	`${c.y}-${pad2(c.m)}-${pad2(c.d)}T23:59`

// localInTzToDate интерпретирует строку datetime-local как wall-clock время в IANA-TZ
// и возвращает соответствующий момент (Date). Нужен, чтобы отправить корректный
// RFC3339-инстант конца рабочего дня в календарной таймзоне.
export const localInTzToDate = (localStr: string, tz: string): Date => {
	const [dp, tp] = localStr.split('T')
	const [y, mo, d] = dp.split('-').map(Number)
	const [h, mi] = (tp || '00:00').split(':').map(Number)
	const want = Date.UTC(y, mo - 1, d, h, mi, 0, 0)
	let guess = want
	for (let i = 0; i < 3; i++) {
		const parts = new Intl.DateTimeFormat('en-US', {
			timeZone: tz,
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit',
			hour12: false,
		}).formatToParts(new Date(guess))
		const get = (type: string) => {
			const v = parts.find(p => p.type === type)?.value
			return v === undefined ? 0 : Number(v)
		}
		const hour = get('hour') % 24
		const asUtc = Date.UTC(get('year'), get('month') - 1, get('day'), hour, get('minute'), get('second'))
		guess += want - asUtc
	}
	return new Date(guess)
}

// dueDateValueForSubmit превращает значение поля datetime-local в строку для API.
// При известной календарной TZ интерпретируем wall-clock в ней и отдаём RFC3339,
// чтобы инстант совпал с бэкендом; иначе оставляем как есть (текущее поведение).
export const dueDateValueForSubmit = (localStr: string, calendar?: PluginCalendar | null): string => {
	if (!localStr) return localStr
	const tz = calendar?.timezone?.trim()
	if (!tz || tz.toLowerCase() === 'local' || !isValidTimezone(tz)) return localStr
	return localInTzToDate(localStr, tz).toISOString()
}

// autoDueDateLocal считает автосрок в рабочих днях от now и возвращает значение
// для <input type="datetime-local"> (конец целевого рабочего дня в календарной TZ).
// days == null/undefined — нет автосрока (null).
export const autoDueDateLocal = (
	days: number | null | undefined,
	calendar?: PluginCalendar | null,
): string | null => {
	if (days == null) return null
	return toDatetimeLocal(addWorkingDaysEnd(new Date(), days, calendar))
}
