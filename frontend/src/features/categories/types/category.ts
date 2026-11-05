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
	defaultDueDays: number | null
	// sortOrder — глобальная позиция категории в реалме. Порядок разделов
	// выводится из него: блок раздела стоит там, где стоит его первая
	// категория. Меняется переупорядочиванием (см. reorderCategories).
	sortOrder: number
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
	defaultDueDays: number | null
}
