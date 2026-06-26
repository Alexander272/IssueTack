import type { IUserShort } from '@/features/user/types/user'

export type TicketStatus = 'open' | 'in_progress' | 'pending' | 'on_hold' | 'resolved' | 'closed' | 'cancelled'

export type Priority = 'low' | 'medium' | 'high' | 'urgent'

export interface ISiteShort {
	id: string
	name: string
}

export interface ICategoryShort {
	id: string
	name: string
}

export interface IGroupShort {
	id: string
	name: string
}

export interface IAttachment {
	id: string
	entityType: string
	entityId: string
	fileName: string
	fileSize: number
	mimeType: string
	uploadedBy: string
	createdAt: string
}

export interface ITask {
	id: string
	title: string
	description: string
	status: TicketStatus
	priority: Priority
	ticketNumber?: number
	realmId?: string
	site: ISiteShort
	category: ICategoryShort
	creator: IUserShort
	owner: IUserShort | null
	group: IGroupShort | null
	assignee: IUserShort | null
	manager: IUserShort | null
	dueDate: string | null
	closedAt: string | null
	createdAt: string
	updatedAt: string
	subtasks?: ISubtask[]
	attachments?: IAttachment[]
}

export interface ITaskFilter {
	number?: number
	siteIds?: string[]
	statuses?: TicketStatus[]
	ownerId?: string
	assigneeId?: string
	search?: string
	priorities?: Priority[]
	dueDateFrom?: string
	dueDateTo?: string
	sort?: string
	limit?: number
	offset?: number
	mode?: string
}

export interface ITaskDTO {
	id: string | null
	title: string
	description: string
	status: TicketStatus
	priority: Priority
	realmId: string
	siteId: string
	categoryId: string
	creatorId: string
	ownerId?: string | null
	groupId?: string | null
	assigneeId?: string | null
	managerId?: string | null
	dueDate?: string | null
	closedAt?: string | null
}

export interface ISubtask {
	id: string
	ticketId: string
	title: string
	description: string
	status: TicketStatus
	priority: Priority
	assignee: IUserShort | null
	dueDate: string | null
	closedAt: string | null
	sortOrder: number
	createdAt: string
	updatedAt: string
}

export interface ISubtaskDTO {
	id: string
	ticketId: string
	title: string
	description?: string
	status: TicketStatus
	priority: Priority
	assigneeId?: string | null
	dueDate?: string | null
	sortOrder: number
}
