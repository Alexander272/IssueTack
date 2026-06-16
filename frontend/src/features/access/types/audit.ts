export interface IAuditLog {
	id: string
	changedBy: string
	changedByName: string
	action: string
	entityType: string
	entity?: string
	entityId?: string
	oldValues?: JSON
	newValues?: JSON
	createdAt: string
}
