import type { TicketStatus, Priority } from '../types/task'

export interface StatusMapValue {
  label: string
  dotColor: string
  bgColor: string
  textColor: string
}

export const STATUS_MAP: Record<TicketStatus, StatusMapValue> = {
  open: { label: 'Новая', dotColor: '#3b82f6', bgColor: '#dbeafe', textColor: '#1e40af' },
  in_progress: { label: 'В работе', dotColor: '#f59e0b', bgColor: '#fef3c7', textColor: '#92400e' },
  pending: { label: 'Ожидание', dotColor: '#8b5cf6', bgColor: '#f3e8ff', textColor: '#6b21a8' },
  on_hold: { label: 'Отложена', dotColor: '#8b5cf6', bgColor: '#f3e8ff', textColor: '#6b21a8' },
  resolved: { label: 'Решена', dotColor: '#10b981', bgColor: '#d1fae5', textColor: '#065f46' },
  closed: { label: 'Закрыта', dotColor: '#10b981', bgColor: '#d1fae5', textColor: '#065f46' },
  cancelled: { label: 'Отменена', dotColor: '#ef4444', bgColor: '#fee2e2', textColor: '#b91c1c' },
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
  { value: 'dueDate_asc', label: 'По сроку (возр.)' },
  { value: 'dueDate_desc', label: 'По сроку (убыв.)' },
  { value: 'priority_asc', label: 'По приоритету (возр.)' },
  { value: 'priority_desc', label: 'По приоритету (убыв.)' },
  { value: 'status', label: 'По статусу' },
  { value: 'closedAt_asc', label: 'По дате закрытия (возр.)' },
  { value: 'closedAt_desc', label: 'По дате закрытия (убыв.)' },
] as const

export const GROUP_BY_OPTIONS = [
  { value: 'category', label: 'По категории' },
  { value: 'status', label: 'По статусу' },
  { value: 'priority', label: 'По приоритету' },
  { value: 'assignee', label: 'По назначению' },
  { value: 'creator', label: 'По заказчику' },
  { value: 'dueDate', label: 'По сроку (неделям)' },
] as const

export type GroupByField = (typeof GROUP_BY_OPTIONS)[number]['value']
