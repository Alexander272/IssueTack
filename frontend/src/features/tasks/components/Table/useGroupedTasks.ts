import { useMemo } from 'react'

import type { ITask } from '../../types/task'
import type { TaskListViewState } from './taskView'
import { getGroupValue, sortGroupKeys } from './getGroupValue'

/** Группировка одна для десктопной таблицы и мобильных карточек. */
export const useGroupedTasks = (tasks: ITask[], view: TaskListViewState) =>
	useMemo(() => {
		const groups: Record<string, ITask[]> = {}
		tasks.forEach(task => {
			const key = getGroupValue(task, view.groupBy, view.hasNoGroups)
			if (!groups[key]) groups[key] = []
			groups[key].push(task)
		})

		return { groups, sortedKeys: sortGroupKeys(Object.keys(groups), view.groupBy) }
	}, [tasks, view.groupBy, view.hasNoGroups])
