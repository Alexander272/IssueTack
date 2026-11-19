import { useMemo } from 'react'

import type { GroupByField } from '../../constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { getHasNoGroups } from '@/features/user/userSlice'

/** Единственный источник набора колонок: по нему рисуются и шапка, и ячейки. */
export const TASK_COLUMNS = [
	{ field: 'ticketNumber', label: '№', sortable: true, width: 70 },
	{ field: 'title', label: 'Тема', sortable: true },
	{ field: 'owner', label: 'Заказчик', sortable: true, width: 180 },
	{ field: 'site', label: 'Площадка', sortable: true, width: 180 },
	{ field: 'createdAt', label: 'Создана', sortable: true, width: 130 },
	{ field: 'dueDate', label: 'Срок', sortable: true, width: 140 },
	{ field: 'closedAt', label: 'Закрыта', sortable: true, width: 140 },
	{ field: 'priority', label: 'Приоритет', sortable: true, width: 130 },
	{ field: 'assignee', label: 'Исполнитель', sortable: true, width: 180 },
	{ field: 'status', label: 'Статус', sortable: true, width: 130 },
	{ field: 'subtasks', label: 'Подзадачи', sortable: false, width: 100 },
] as const

export type TaskColumnField = (typeof TASK_COLUMNS)[number]['field']

/** Описание колонки для шапки: ширина необязательна (у «Темы» её нет). */
export type TaskColumn = {
	field: TaskColumnField
	label: string
	sortable: boolean
	width?: number
}

// Колонки, скрываемые у «обычного пользователя» без групп: срок и приоритет ему не нужны.
const GROUPLESS_HIDDEN_FIELDS = new Set<TaskColumnField>(['dueDate', 'priority'])

type ColumnInput = {
	isArchive: boolean
	groupBy: GroupByField
	groupEnabled: boolean
	hasNoGroups: boolean
}

/** Видимые колонки считаются один раз и используются всеми уровнями таблицы. */
export const visibleTaskColumns = ({ isArchive, groupBy, groupEnabled, hasNoGroups }: ColumnInput) =>
	new Set<TaskColumnField>(
		TASK_COLUMNS.filter(col => {
			// Разделяем колонки для обычного режима и архива
			if (isArchive && col.field === 'dueDate') return false
			if (!isArchive && col.field === 'closedAt') return false

			// Скрываем колонки, ненужные обычному пользователю
			if (hasNoGroups && GROUPLESS_HIDDEN_FIELDS.has(col.field)) return false

			// Скрываем колонку, по которой идёт активная группировка —
			// заголовок группы уже несёт эту информацию
			if (groupEnabled && col.field === groupBy) return false
			return true
		}).map(col => col.field),
	)

/**
 * view — один объект с решениями о представлении списка. Он заменяет набор
 * разрозненных пропов (isArchive/groupBy/groupEnabled/isMobile-флаги), из-за
 * которых шапка и ячейки могли по-разному решать, что показывать.
 */
export interface TaskListViewState extends ColumnInput {
	sort: string
	onSortChange: (sort: string) => void
	columns: ReadonlySet<TaskColumnField>
}

export const useTaskListView = (
	params: Omit<TaskListViewState, 'columns' | 'hasNoGroups'>,
): TaskListViewState => {
	const hasNoGroups = useAppSelector(getHasNoGroups)
	const { isArchive, groupBy, groupEnabled, sort, onSortChange } = params
	const columns = useMemo(
		() => visibleTaskColumns({ isArchive, groupBy, groupEnabled, hasNoGroups }),
		[isArchive, groupBy, groupEnabled, hasNoGroups],
	)

	return useMemo(
		() => ({ isArchive, groupBy, groupEnabled, sort, onSortChange, hasNoGroups, columns }),
		[isArchive, groupBy, groupEnabled, sort, onSortChange, hasNoGroups, columns],
	)
}
