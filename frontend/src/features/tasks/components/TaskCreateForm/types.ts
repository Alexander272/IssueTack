import type { Priority } from '../../types/task'

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
}

export type Props = {
	onSuccess?: () => void
	onCancel?: () => void
	embedded?: boolean
}
