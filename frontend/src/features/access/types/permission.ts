export interface IPermission {
	id: string
	object: string
	action: string
	description: string
}

export interface IGroupedPermission {
	group: string
	title: string
	items: IPermission[]
}
