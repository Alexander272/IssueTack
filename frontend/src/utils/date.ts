import dayjs from 'dayjs'
// import relativeTime from 'dayjs/plugin/relativeTime'
import calendar from 'dayjs/plugin/calendar'
import 'dayjs/locale/ru'

// dayjs.extend(relativeTime)
dayjs.extend(calendar)
dayjs.locale('ru')

export const getSmartDate = (date: string) => {
	if (!date) {
		return '-'
	}

	// const now = dayjs()
	const target = dayjs(date)
	// const diffInDays = now.diff(target, 'day')

	if (target.year() == 1970) return '-'

	// Если прошло больше 1 дня (но меньше месяца), используем "X дней назад"
	// if (diffInDays > 1 && diffInDays < 30) {
	// 	return target.fromNow()
	// }

	const fullFormat = 'dddd, DD MMM YYYY HH:mm'

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
