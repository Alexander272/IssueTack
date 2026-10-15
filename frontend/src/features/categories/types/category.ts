import type { Priority } from '@/features/tasks/types/task'

export interface ICategory {
	id: string
	name: string
	description: string
	groupId: string
	categoryGroupId: string | null
	categoryGroup?: ICategoryGroupShort
	priority: Priority
	isActive: boolean
	isHidden: boolean
	createdAt: string
	updatedAt: string
}

export interface ICategoryGroupShort {
	id: string
	name: string
	isHidden?: boolean
}

export interface ICategoryDTO {
	id: string | null
	name: string
	description: string
	groupId: string
	categoryGroupId: string | null
	priority: Priority
	isActive: boolean
	isHidden: boolean
}
