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
	/** Подкатегории категории: параметры разово применяются к новой заявке. */
	modifiers?: ICategoryModifier[]
	createdAt: string
	updatedAt: string
}

/** Подкатегория (модификатор категории). Пустой defaultDueDays/priority = наследовать категорию. */
export interface ICategoryModifier {
	id: string
	realmId: string
	categoryId: string
	name: string
	description: string
	defaultDueDays: number | null
	priority: Priority | null
	isActive: boolean
	sortOrder: number
	createdAt: string
	updatedAt: string
}

export interface ICategoryModifierDTO {
	id: string | null
	categoryId: string
	name: string
	description: string
	defaultDueDays: number | null
	priority: Priority | null
	isActive: boolean
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
