import type { ComponentType } from 'react'
import type { SvgIconProps } from '@mui/material'
import { ZapIcon, CalendarDaysIcon, CalendarRangeIcon, ClockIcon, HourglassIcon } from 'lucide-mui'

// Значения по умолчанию и пресеты порогов напоминаний (в минутах до дедлайна).
// Держим синхронно с бэкендом: models.DefaultDeadlineReminders() = [2880, 1440, 360].
export const DEFAULT_REMINDERS = [2880, 1440, 360]

export interface ReminderOption {
	value: number
	label: string
}

export const REMINDER_OPTIONS: ReminderOption[] = [
	{ value: 30, label: 'за 30 минут' },
	{ value: 60, label: 'за 1 час' },
	{ value: 180, label: 'за 3 часа' },
	{ value: 360, label: 'за 6 часов' },
	{ value: 720, label: 'за 12 часов' },
	{ value: 1440, label: 'за 1 день' },
	{ value: 2880, label: 'за 2 дня' },
	{ value: 4320, label: 'за 3 дня' },
	{ value: 10080, label: 'за 7 дней' },
	{ value: 20160, label: 'за 14 дней' },
]

const REMINDER_LABELS = new Map(REMINDER_OPTIONS.map(option => [option.value, option.label]))

export const getReminderLabel = (minutes: number): string => {
	const label = REMINDER_LABELS.get(minutes)
	return label ?? `за ${minutes} мин`
}

export interface ReminderVisual {
	Icon: ComponentType<SvgIconProps>
	bg: string
	color: string
}

// Цвет и иконка зависят от того, насколько заранее напоминаем: чем ближе к дедлайну, тем «тревожнее».
export const getReminderVisual = (minutes: number): ReminderVisual => {
	if (minutes <= 60) {
		return { Icon: ZapIcon, bg: 'rgba(239,68,68,0.12)', color: '#dc2626' }
	}
	if (minutes <= 720) {
		return { Icon: HourglassIcon, bg: 'rgba(249,115,22,0.12)', color: '#ea580c' }
	}
	if (minutes <= 1440) {
		return { Icon: ClockIcon, bg: 'rgba(245,158,11,0.14)', color: '#d97706' }
	}
	if (minutes <= 4320) {
		return { Icon: CalendarDaysIcon, bg: 'rgba(59,130,246,0.12)', color: '#3b82f6' }
	}
	return { Icon: CalendarRangeIcon, bg: 'rgba(99,102,241,0.12)', color: '#6366f1' }
}

// По убыванию: сначала самые «ранние» напоминания (за 14 дней), в конце — близкие к дедлайну.
export const sortReminders = (reminders: number[]): number[] => [...new Set(reminders)].sort((a, b) => b - a)
