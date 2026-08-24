export interface IActivityLogActor {
	id: string
	firstName?: string
	lastName?: string
}

export interface IActivityLog {
	id: string
	action: string
	changedBy: string
	changedByName: string
	entityType: string
	entityId: string
	entity: string
	parentId?: string
	oldValues?: Record<string, string>
	newValues?: Record<string, string>
	createdAt: string
	actor?: IActivityLogActor
}
