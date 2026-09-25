import type { Priority } from '../../types/task'

export interface SubtaskFormValues {
	title: string
	description: string
}

export interface FormValues {
	title: string
	description: string
	priority: Priority
	categoryId: string
	groupId: string | null
	ownerId: string | null
	assigneeId: string | null
	siteId: string
	dueDate: string | null
	subtasks: SubtaskFormValues[]
}

export type Props = {
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
	onSavingChange?: (saving: boolean) => void
}
