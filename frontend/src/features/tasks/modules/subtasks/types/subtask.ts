import type { IUserShort } from '@/features/user/types/user'
import type { TicketStatus, Priority } from '../../../types/task'

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
