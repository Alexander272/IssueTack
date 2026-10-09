import dayjs from 'dayjs'
// import relativeTime from 'dayjs/plugin/relativeTime'
import calendar from 'dayjs/plugin/calendar'
import 'dayjs/locale/ru'

// dayjs.extend(relativeTime)
dayjs.extend(calendar)
dayjs.locale('ru')

export type DueDateTone = 'overdue' | 'today' | 'soon' | null

export const getShortDate = (date: string | null) => {
	if (!date) {
		return '—'
	}

	const target = dayjs(date)
	const now = dayjs()

	// Проверяем, совпадает ли год с текущим
	if (target.year() === now.year()) {
		return target.format('D MMM') // Текущий год: "9 окт"
	} else {
		return target.format('D MMM YY') // Другой год: "9 окт 25"
	}
}

export const getDate = (date: string) => {
	if (!date) {
		return '—'
	}

	return dayjs(date).format('DD.MM.YYYY')
}

export const getDueDateTone = (dueDate: string | null): DueDateTone => {
	if (!dueDate) return null

	const target = dayjs(dueDate).startOf('day')
	const now = dayjs().startOf('day')
	const diff = target.diff(now, 'day')

	if (diff < 0) return 'overdue'
	if (diff === 0) return 'today'
	if (diff <= 3) return 'soon'
	return null
}

export const getSmartDate = (date: string) => {
	if (!date) {
		return '—'
	}

	// const now = dayjs()
	const target = dayjs(date)
	// const diffInDays = now.diff(target, 'day')

	if (target.year() == 1970) return '-'

	// Если прошло больше 1 дня (но меньше месяца), используем "X дней назад"
	// if (diffInDays > 1 && diffInDays < 30) {
	// 	return target.fromNow()
	// }

	const fullFormat = 'dd, DD MMM YYYY HH:mm'

	// Для сегодня, вчера и совсем старых дат — календарный формат
	return target.calendar(null, {
		sameDay: '[Сегодня в] HH:mm',
		lastDay: '[Вчера в] HH:mm',
		nextDay: fullFormat,
		nextWeek: fullFormat,
		lastWeek: fullFormat,
		sameElse: fullFormat,
	})
}
