import type { IUserShort } from '@/features/user/types/user'

export interface IGroup {
	id: string
	name: string
	description: string
	createdAt: string
	updatedAt: string
	members?: IUserShort[]
	defaultAssigneeId?: string | null
	managerId?: string | null
	defaultAssignee?: IUserShort
	manager?: IUserShort
}

export interface IGroupDTO {
	id?: string
	name: string
	description: string
	managerId: string | null
	defaultAssigneeId: string | null
	memberIds: string[]
}
