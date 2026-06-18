export interface IRealm {
	id: string
	name: string
	slug: string
	description: string
	isActive: boolean
	userCount: number
	createdAt: string
}

export interface IRealmDTO {
	id: string
	name: string
	slug: string
	description: string
	isActive: boolean
}
