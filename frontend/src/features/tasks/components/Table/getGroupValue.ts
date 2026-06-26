import type { ITask } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'
import { STATUS_MAP, PRIORITY_MAP } from '../../constants/taskMaps'

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
