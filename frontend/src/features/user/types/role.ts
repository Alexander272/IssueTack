export interface IRole {
	id: string
	slug: string
	name: string
	description: string
	level: number
	isActive: boolean
	isSystem: boolean
	isEditable: boolean
	createdAt: Date
	updatedAt: Date
}

export interface IRoleDTO {
	id: string
	slug: string
	name: string
	description: string
	level: number
}

export interface IRoleWithStats extends IRole {
	children: string[]
	inheritance: { [key: string]: string[] }
	perms: IPermsCount
	userCount: number
}

export interface IPermsCount {
	own: IPerm
	inherited: IPerm
	total: IPerm
}
interface IPerm {
	items: string[]
	count: number
}

export interface IFullRole {
	id: string
	name: string
	description: string
	level: number
	extends: string[]
	isShow: boolean
}

export interface IRolePermissionItem {
	permissionId: string
	object: string
	action: string
	isAssigned: boolean
	isInherited: boolean
	status?: 'original' | 'changed' | 'new'
}

export interface IRolePermissionsGrouped {
	group: string
	resources: IRolePermissionItem[]
}

export interface IRoleWithPerms extends IRole {
	extends?: string[]
	perms: IRolePermissionsGrouped[]
}

export interface IRoleWithPermsDTO extends IRoleDTO {
	permissions: string[]
	inherits: string[]
}
