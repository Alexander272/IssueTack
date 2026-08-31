import type { ITask } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'
import { STATUS_MAP, PRIORITY_MAP, PRIORITY_ORDER } from '../../constants/taskMaps'

export function getGroupValue(task: ITask, groupBy: GroupByField): string {
	switch (groupBy) {
		case 'category':
			return task.category.name
		case 'status':
			return STATUS_MAP[task.status]?.label ?? task.status
		case 'priority':
			return PRIORITY_MAP[task.priority]?.label ?? task.priority
		case 'site':
			return task.site?.name || 'Без площадки'
		case 'assignee': {
			if (task.assignee) return `👤 ${task.assignee.lastName} ${task.assignee.firstName}`
			if (task.group) return `👥 ${task.group.name}`
			return 'Без назначения'
		}
		case 'creator':
			return task.creator ? `${task.owner?.lastName} ${task.owner?.firstName}` : 'Без заказчика'
		case 'dueDate': {
			if (!task.dueDate) return 'Без срока'
			const date = new Date(task.dueDate)
			const week = Math.ceil(
				(date.getTime() - new Date(date.getFullYear(), 0, 1).getTime()) / (7 * 24 * 60 * 60 * 1000),
			)
			return `${date.getFullYear()} неделя ${week}`
		}
		default:
			return task.category.name
	}
}

const priorityRankByLabel = new Map(PRIORITY_ORDER.map(p => [PRIORITY_MAP[p].label, p]))

export function sortGroupKeys(keys: string[], groupBy: GroupByField): string[] {
	if (groupBy !== 'priority') return [...keys].sort((a, b) => a.localeCompare(b))
	const rank = (key: string) => {
		const p = priorityRankByLabel.get(key)
		return p === undefined ? PRIORITY_ORDER.length : PRIORITY_ORDER.indexOf(p)
	}
	return [...keys].sort((a, b) => rank(a) - rank(b))
}
