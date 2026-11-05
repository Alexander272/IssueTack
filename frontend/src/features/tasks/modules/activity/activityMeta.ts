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

// Вид значения поля — от него зависит рендер в таймлайне:
// status/priority — чипы из STATUS_MAP/PRIORITY_MAP; diff — зачёркнутое старое → жирное новое;
// date — дата; plain — обычный текст (ФИО/имя группы/площадки). У created тело — просто «заголовок».
export type FieldKind = 'status' | 'priority' | 'diff' | 'date' | 'plain'

export interface FieldMeta {
	/** Название поля — показывается только когда в записи менялось несколько полей. */
	label: string
	/** Фраза действия, когда изменение единственное: «изменил статус». */
	phrase: string
	icon: FC<SvgIconProps>
	/** Цвет rail-кружка (не текста). */
	color: string
	kind: FieldKind
}

const meta = (
	label: string,
	phrase: string,
	icon: FC<SvgIconProps>,
	color: string,
	kind: FieldKind,
): FieldMeta => ({ label, phrase, icon, color, kind })

// Ключи — теги изменений (`GetChanges`) плюс ключи снапшота удаления (`Delete` кладёт
// их без суффикса `_changed`: title/status/priority/assignee).
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
	closed: meta('Закрытие', 'изменил дату закрытия', CalendarClock, '#6b7280', 'date'),
	site_changed: meta('Площадка', 'изменил площадку', Building2, '#14b8a6', 'plain'),
	category_changed: meta('Категория', 'изменил категорию', Layers, '#a855f7', 'plain'),
	// снапшот удаления
	title: meta('Заголовок', 'изменил заголовок', Pencil, '#6b7280', 'diff'),
	description: meta('Описание', 'изменил описание', Pencil, '#6b7280', 'diff'),
	status: meta('Статус', 'изменил статус', ArrowRightLeft, '#3b82f6', 'status'),
	priority: meta('Приоритет', 'изменил приоритет', Flag, '#f59e0b', 'priority'),
	assignee: meta('Исполнитель', 'изменил исполнителя', UserRound, '#8b5cf6', 'plain'),
}

// Если тег незнаком (новое поле на бэкенде) — не роняем рендер, показываем как текст.
export const fieldMeta = (tag: string): FieldMeta =>
	FIELD_META[tag] ?? meta(tag, 'изменил поле', Pencil, '#6b7280', 'plain')

// Общая фраза/иконка для записи, где менялось сразу несколько полей.
export const MULTI_META = { phrase: 'изменил поля', icon: Pencil, color: '#6b7280' } as const

export const ACTION_META = {
	created: { phrase: 'создал заявку', icon: Plus, color: '#10b981' },
	deleted: { phrase: 'удалил заявку', icon: Trash2, color: '#ef4444' },
} as const

// Поля, которые «обычному пользователю» без групп не показываем: приоритет и срок он
// не видит нигде (Meta/InfoBar/фильтры), группа и руководитель ему ни о чём не говорят.
// Если у изменения не остаётся ни одного видимого поля, весь лог-вход скрывается.
export const GROUPLESS_HIDDEN_FIELDS = new Set<string>([
	'priority_changed',
	'priority',
	'due_date_changed',
	'group_changed',
	'group_assigned',
	'manager_changed',
])
