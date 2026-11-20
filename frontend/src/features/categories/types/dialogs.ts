import type { ICategory, ICategoryDTO, ICategoryModifier, ICategoryModifierDTO } from './category'
import type { ICategoryGroup, ICategoryGroupDTO } from './categoryGroup'

/** Единственный «контракт» таблицы с диалогами. */
export type CategoryDialogRequest =
	| { kind: 'create-category'; sectionId?: string | null }
	| { kind: 'edit-category'; category: ICategory }
	| { kind: 'create-section' }
	| { kind: 'edit-section'; section: ICategoryGroup }
	| { kind: 'create-modifier'; categoryId?: string }
	| { kind: 'edit-modifier'; modifier: ICategoryModifier }

export type CategoryDialogState =
	| { kind: 'category'; data: ICategoryDTO }
	| { kind: 'section'; data: ICategoryGroupDTO }
	| { kind: 'modifier'; data: ICategoryModifierDTO }
	| null
