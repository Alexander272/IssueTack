export interface ICategoryGroup {
	id: string
	realmId: string
	name: string
	description: string
	sortOrder: number
	isHidden: boolean
	createdAt: string
	updatedAt: string
}

export interface ICategoryGroupDTO {
	id: string | null
	name: string
	description: string
	sortOrder: number
	isHidden: boolean
}
