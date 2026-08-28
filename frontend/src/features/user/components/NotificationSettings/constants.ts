import { ClockIcon, MessageSquareIcon, PlusCircleIcon, RepeatIcon } from 'lucide-mui'

export const CATEGORY_EVENTS = [
	{ key: 'newTask', label: 'Новая задача', Icon: PlusCircleIcon, color: '#3b82f6' },
	{ key: 'status', label: 'Изменение статуса', Icon: RepeatIcon, color: '#f59e0b' },
	{ key: 'comment', label: 'Комментарий', Icon: MessageSquareIcon, color: '#10b981' },
	{ key: 'overdue', label: 'Просрочена', Icon: ClockIcon, color: '#ef4444' },
] as const

export const GROUP_EVENTS = [
	{ key: 'newTask', label: 'Новые задачи' },
	{ key: 'overdue', label: 'Просроченные' },
] as const

export type CategoryEventKey = (typeof CATEGORY_EVENTS)[number]['key']
export type GroupEventKey = (typeof GROUP_EVENTS)[number]['key']
