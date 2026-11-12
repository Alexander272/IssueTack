import type { FC } from 'react'
import type { SvgIconProps } from '@mui/material'
import {
	ArrowRightLeft,
	Flag,
	Pencil,
	CalendarClock,
	UserRound,
	UserCog,
	Users,
	Building2,
	Layers,
	Plus,
	Trash2,
} from 'lucide-mui'

export type FieldKind = 'status' | 'priority' | 'diff' | 'date' | 'plain'

export interface FieldMeta {
	label: string
	phrase: string
	icon: FC<SvgIconProps>
	color: string
	kind: FieldKind
}

const meta = (label: string, phrase: string, icon: FC<SvgIconProps>, color: string, kind: FieldKind): FieldMeta => ({
	label,
	phrase,
	icon,
	color,
	kind,
})

export const FIELD_META: Record<string, FieldMeta> = {
	title_changed: meta('Заголовок', 'изменил заголовок', Pencil, '#6b7280', 'diff'),
	description_changed: meta('Описание', 'изменил описание', Pencil, '#6b7280', 'diff'),
	status_changed: meta('Статус', 'изменил статус', ArrowRightLeft, '#3b82f6', 'status'),
	priority_changed: meta('Приоритет', 'изменил приоритет', Flag, '#f59e0b', 'priority'),
	assigned: meta('Исполнитель', 'назначил исполнителя', UserRound, '#8b5cf6', 'plain'),
	assign_changed: meta('Исполнитель', 'изменил исполнителя', UserRound, '#8b5cf6', 'plain'),
	owner_changed: meta('Заказчик', 'изменил заказчика', UserRound, '#8b5cf6', 'plain'),
	manager_changed: meta('Руководитель', 'изменил руководителя', UserCog, '#8b5cf6', 'plain'),
	group_changed: meta('Группа', 'изменил группу', Users, '#0ea5e9', 'plain'),
	group_assigned: meta('Группа', 'назначил группу', Users, '#0ea5e9', 'plain'),
	due_date_changed: meta('Срок', 'изменил срок', CalendarClock, '#f59e0b', 'date'),
	due_date_requested: meta('Срок', 'запросил перенос', CalendarClock, '#f59e0b', 'date'),
	due_date_rejected: meta('Срок', 'не перенёс', CalendarClock, '#ef4444', 'date'),
	closed: meta('Закрытие', 'изменил дату закрытия', CalendarClock, '#6b7280', 'date'),
	site_changed: meta('Площадка', 'изменил площадку', Building2, '#14b8a6', 'plain'),
	category_changed: meta('Категория', 'изменил категорию', Layers, '#a855f7', 'plain'),
	title: meta('Заголовок', 'изменил заголовок', Pencil, '#6b7280', 'diff'),
	status: meta('Статус', 'изменил статус', ArrowRightLeft, '#3b82f6', 'status'),
	priority: meta('Приоритет', 'изменил приоритет', Flag, '#f59e0b', 'priority'),
	assignee: meta('Исполнитель', 'изменил исполнителя', UserRound, '#8b5cf6', 'plain'),
}

export const fieldMeta = (tag: string): FieldMeta =>
	FIELD_META[tag] ?? meta(tag, 'изменил поле', Pencil, '#6b7280', 'plain')

export const MULTI_META = { phrase: 'изменил поля', icon: Pencil, color: '#6b7280' } as const
export const MULTI_META_SUBTASK = { phrase: 'изменил подзадачу', icon: Pencil, color: '#6b7280' } as const

export const ACTION_META = {
	created: {
		ticket: { phrase: 'создал заявку', icon: Plus, color: '#10b981' },
		subtask: { phrase: 'создал подзадачу', icon: Plus, color: '#10b981' },
	},
	deleted: {
		ticket: { phrase: 'удалил заявку', icon: Trash2, color: '#ef4444' },
		subtask: { phrase: 'удалил подзадачу', icon: Trash2, color: '#ef4444' },
	},
} as const

export const GROUPLESS_HIDDEN_FIELDS = new Set<string>([
	'priority_changed',
	'priority',
	'due_date_changed',
	'due_date_requested',
	'due_date_rejected',
	'group_changed',
	'group_assigned',
	'manager_changed',
])
