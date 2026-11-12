import { AlarmClockIcon, CalendarClockIcon, ClockIcon, FlameIcon, MessageSquareIcon, PlusCircleIcon, RepeatIcon } from 'lucide-mui'

export const CATEGORY_EVENTS = [
	{ key: 'newTask', label: 'Новая задача', Icon: PlusCircleIcon, color: '#3b82f6' },
	{ key: 'status', label: 'Изменение статуса', Icon: RepeatIcon, color: '#f59e0b' },
	{ key: 'comment', label: 'Комментарий', Icon: MessageSquareIcon, color: '#10b981' },
	{ key: 'overdue', label: 'Просрочена', Icon: ClockIcon, color: '#ef4444' },
	{ key: 'reminderHigh', label: 'Напоминание (высокий)', Icon: AlarmClockIcon, color: '#f97316' },
	{ key: 'reminderUrgent', label: 'Напоминание (критичный)', Icon: FlameIcon, color: '#dc2626' },
	{ key: 'dueDateRequested', label: 'Запрос переноса срока', Icon: CalendarClockIcon, color: '#f59e0b' },
	{ key: 'dueDateApproved', label: 'Перенос утверждён', Icon: CalendarClockIcon, color: '#10b981' },
	{ key: 'dueDateRejected', label: 'Перенос отклонён', Icon: CalendarClockIcon, color: '#ef4444' },
] as const

export type CategoryEventKey = (typeof CATEGORY_EVENTS)[number]['key']
