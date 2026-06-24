import type { Priority } from '@/features/tasks/types/task'

export interface ICategory {
	id: string
	name: string
	description: string
	groupId: string
	priority: Priority
	isActive: boolean
	createdAt: string
	updatedAt: string
}

export interface ICategoryDTO {
	id: string | null
	name: string
	description: string
	groupId: string
	priority: Priority
	isActive: boolean
}
