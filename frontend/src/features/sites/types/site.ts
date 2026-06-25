export interface ISite {
	id: string
	name: string
	address: string
	createdAt: string
	updatedAt: string
}

export interface ISiteDTO {
	id: string | null
	name: string
	address: string
}
