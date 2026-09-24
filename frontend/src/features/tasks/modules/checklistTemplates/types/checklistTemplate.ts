export interface IChecklistTemplate {
	id: string
	realmId: string
	title: string
	description: string
	/** Автор шаблона: без права checklist:read видны только свои шаблоны. */
	createdBy?: string | null
	createdAt?: string
	updatedAt?: string
}

export interface IChecklistTemplateItem {
	id: string
	templateId: string
	title: string
	description: string
	sortOrder: number
}

export interface IChecklistTemplateDTO {
	id?: string
	realmId?: string
	title: string
	description?: string
}

export interface IChecklistTemplateItemDTO {
	id?: string
	templateId?: string
	title: string
	sortOrder?: number
}