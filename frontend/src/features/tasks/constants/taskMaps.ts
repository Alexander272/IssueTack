import type { FC } from 'react'
import type { SvgIconProps } from '@mui/material'
import type { TicketStatus, Priority } from '../types/task'
import { PlayIcon } from '@/components/Icons/PlayIcon'
import { ClockIcon } from '@/components/Icons/ClockIcon'
import { PauseIcon } from '@/components/Icons/PauseIcon'
import { CheckIcon } from '@/components/Icons/CheckIcon'
import { LockIcon } from '@/components/Icons/LockIcon'
import { CloseRoundIcon } from '@/components/Icons/CloseRoundIcon'
import { WarnIcon } from '@/components/Icons/WarnIcon'

export interface StatusMapValue {
	label: string
	icon: FC<SvgIconProps>
	bgColor: string
	textColor: string
}

// export const STATUS_MAP: Record<TicketStatus, StatusMapValue> = {
// 	open: { label: 'Новая', icon: WarnIcon, bgColor: '#dbeafe', textColor: '#1e40af' },
// 	in_progress: { label: 'В работе', icon: PlayIcon, bgColor: '#fef3c7', textColor: '#92400e' },
// 	pending: { label: 'Ожидание', icon: ClockIcon, bgColor: '#f3e8ff', textColor: '#6b21a8' },
// 	on_hold: { label: 'Отложена', icon: PauseIcon, bgColor: '#f3e8ff', textColor: '#6b21a8' },
// 	resolved: { label: 'Решена', icon: CheckIcon, bgColor: '#d1fae5', textColor: '#065f46' },
// 	closed: { label: 'Закрыта', icon: LockIcon, bgColor: '#d1fae5', textColor: '#065f46' },
// 	cancelled: { label: 'Отменена', icon: CloseRoundIcon, bgColor: '#fee2e2', textColor: '#b91c1c' },
// }
export const STATUS_MAP: Record<TicketStatus, StatusMapValue> = {
	open: { label: 'Новая', icon: WarnIcon, bgColor: '#E1F5FE', textColor: '#01579B' },
	in_progress: { label: 'В работе', icon: PlayIcon, bgColor: '#FFF3E0', textColor: '#E65100' },
	pending: { label: 'Ожидание', icon: ClockIcon, bgColor: '#FFFDE7', textColor: '#F57F17' },
	on_hold: { label: 'Отложена', icon: PauseIcon, bgColor: '#F3E5F5', textColor: '#4A148C' },
	resolved: { label: 'Решена', icon: CheckIcon, bgColor: '#E8F5E9', textColor: '#1B5E20' },
	closed: { label: 'Закрыта', icon: LockIcon, bgColor: '#C8E6C9', textColor: '#0D47A1' },
	cancelled: { label: 'Отменена', icon: CloseRoundIcon, bgColor: '#FFEBEE', textColor: '#B71C1C' },
}

export interface PriorityMapValue {
	label: string
	barCount: number
	barColor: string
	bgColor: string
	textColor: string
}

export const PRIORITY_MAP: Record<Priority, PriorityMapValue> = {
	low: { label: 'Низкий', barCount: 1, barColor: '#10b981', bgColor: '#ecfdf5', textColor: '#065f46' },
	medium: { label: 'Средний', barCount: 2, barColor: '#f59e0b', bgColor: '#fef3c7', textColor: '#92400e' },
	high: { label: 'Высокий', barCount: 3, barColor: '#ef4444', bgColor: '#fee2e2', textColor: '#b91c1c' },
	urgent: { label: 'Критичный', barCount: 4, barColor: '#dc2626', bgColor: '#fef2f2', textColor: '#991b1b' },
}

export const QUEUE_OPTIONS = [
	{ value: 'all', label: 'Все мои очереди' },
	{ value: 'personal', label: '👤 Лично мне' },
	{ value: 'group1', label: '👥 IT-поддержка' },
	{ value: 'group2', label: '👥 Сетевая адм.' },
] as const

export const STATUS_OPTIONS: { value: TicketStatus | 'all'; label: string }[] = [
	{ value: 'all', label: 'Все статусы' },
	{ value: 'open', label: 'Новые' },
	{ value: 'in_progress', label: 'В работе' },
	{ value: 'pending', label: 'Ожидание' },
	{ value: 'resolved', label: 'Решены' },
	{ value: 'closed', label: 'Закрыты' },
]

export const SORT_OPTIONS = [
  { value: 'ticketNumber_asc', label: 'По номеру (возр.)' },
  { value: 'ticketNumber_desc', label: 'По номеру (убыв.)' },
  { value: 'title_asc', label: 'По теме (А-Я)' },
  { value: 'title_desc', label: 'По теме (Я-А)' },
  { value: 'dueDate_asc', label: 'По сроку (возр.)' },
  { value: 'dueDate_desc', label: 'По сроку (убыв.)' },
  { value: 'priority_asc', label: 'По приоритету (возр.)' },
  { value: 'priority_desc', label: 'По приоритету (убыв.)' },
  { value: 'status_asc', label: 'По статусу (А-Я)' },
  { value: 'status_desc', label: 'По статусу (Я-А)' },
  { value: 'owner_asc', label: 'По заказчику (А-Я)' },
  { value: 'owner_desc', label: 'По заказчику (Я-А)' },
  { value: 'category_asc', label: 'По категории (А-Я)' },
  { value: 'category_desc', label: 'По категории (Я-А)' },
  { value: 'assignee_asc', label: 'По назначению (А-Я)' },
  { value: 'assignee_desc', label: 'По назначению (Я-А)' },
  { value: 'closedAt_asc', label: 'По дате закрытия (возр.)' },
  { value: 'closedAt_desc', label: 'По дате закрытия (убыв.)' },
] as const

export const GROUP_BY_OPTIONS = [
  { value: 'site', label: 'По площадке' },
  { value: 'status', label: 'По статусу' },
  { value: 'priority', label: 'По приоритету' },
  { value: 'assignee', label: 'По назначению' },
  { value: 'creator', label: 'По заказчику' },
  { value: 'dueDate', label: 'По сроку (неделям)' },
  { value: 'category', label: 'По категории' },
] as const

export type GroupByField = (typeof GROUP_BY_OPTIONS)[number]['value']
