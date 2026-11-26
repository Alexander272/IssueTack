import type { ICategory, ICategoryDTO, ICategoryModifier, ICategoryModifierDTO } from '../types/category'
import type { ICategoryGroup, ICategoryGroupDTO } from '../types/categoryGroup'

export const toCategoryDTO = (c: ICategory): ICategoryDTO => ({
	id: c.id,
	name: c.name,
	description: c.description,
	groupId: c.groupId,
	categoryGroupId: c.categoryGroupId ?? null,
	priority: c.priority,
	isActive: c.isActive,
	isHidden: c.isHidden,
	defaultDueDays: c.defaultDueDays ?? null,
})

export const toGroupDTO = (g: ICategoryGroup): ICategoryGroupDTO => ({
	id: g.id,
	name: g.name,
	description: g.description,
	isHidden: g.isHidden,
})

export const toModifierDTO = (m: ICategoryModifier): ICategoryModifierDTO => ({
	id: m.id,
	categoryId: m.categoryId,
	name: m.name,
	description: m.description,
	keywords: m.keywords ?? [],
	defaultDueDays: m.defaultDueDays,
	priority: m.priority,
	isActive: m.isActive,
})
