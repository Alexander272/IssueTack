export interface ICategoryGroup {
	id: string
	realmId: string
	name: string
	description: string
	sortOrder: number
	createdAt: string
	updatedAt: string
}

export interface ICategoryGroupDTO {
	id: string | null
	name: string
	description: string
	sortOrder: number
}
