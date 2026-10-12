export interface IEquipmentType {
	id: string
	realmId: string
	name: string
	isComponent: boolean
	sortOrder: number
	createdAt: string
	updatedAt: string
}

export interface IEquipmentTypeDTO {
	id?: string
	realmId?: string
	name: string
	isComponent: boolean
	sortOrder: number
}

export type EquipmentStatus =
	| 'in_stock'
	| 'in_use'
	| 'repair'
	| 'reserved'
	| 'written_off'

const EQUIPMENT_STATUS_MAP: Record<EquipmentStatus, { label: string; color: string }> = {
	in_stock: { label: 'На складе', color: '#6b7280' },
	in_use: { label: 'В эксплуатации', color: '#16a34a' },
	repair: { label: 'В ремонте', color: '#ea580c' },
	reserved: { label: 'Зарезервировано', color: '#2563eb' },
	written_off: { label: 'Списано', color: '#dc2626' },
}

export const getEquipmentStatus = (status: EquipmentStatus) =>
	EQUIPMENT_STATUS_MAP[status] ?? { label: status, color: '#6b7280' }

export const EQUIPMENT_STATUS_OPTIONS = Object.entries(EQUIPMENT_STATUS_MAP).map(([value, { label }]) => ({
	value: value as EquipmentStatus,
	label,
}))

export interface IEquipment {
	id: string
	realmId: string
	typeId: string
	typeName: string
	isComponent: boolean
	name: string
	vendor: string
	model: string
	serialNumber: string | null
	inventoryNumber: string | null
	specs: Record<string, unknown>
	status: EquipmentStatus
	parentId: string | null
	siteId: string | null
	siteName: string
	holderUserId: string | null
	holderName: string | null
	holderFullName: string
	location: string | null
	purchaseDate: string | null
	warrantyUntil: string | null
	cost: number | null
	notes: string
	isKit: boolean
	kitId: string | null
	createdBy: string | null
	createdAt: string
	updatedAt: string
}

export interface IEquipmentDTO {
	id?: string
	typeId: string
	name: string
	vendor?: string
	model?: string
	serialNumber?: string | null
	inventoryNumber?: string | null
	specs?: Record<string, unknown>
	status: EquipmentStatus
	parentId?: string | null
	siteId?: string | null
	holderUserId?: string | null
	holderName?: string | null
	location?: string | null
	purchaseDate?: string | null
	warrantyUntil?: string | null
	cost?: number | null
	notes?: string
	isKit?: boolean
	kitId?: string | null
}

export interface IEquipmentFilter {
	typeId?: string
	status?: EquipmentStatus
	siteId?: string
	holderId?: string
	serial?: string
	inventory?: string
	search?: string
	limit?: number
	offset?: number
}

export interface IEquipmentList {
	data: IEquipment[]
	total: number
}

export interface IEquipmentAssignment {
	id: string
	equipmentId: string
	userId: string | null
	holderName: string | null
	siteId: string | null
	location: string | null
	assignedAt: string
	unassignedAt: string | null
	assignedBy: string | null
	comment: string
}

export type EquipmentEventType =
	| 'created'
	| 'updated'
	| 'assigned'
	| 'unassigned'
	| 'moved'
	| 'status_changed'
	| 'component_added'
	| 'component_removed'
	| 'component_replaced'
	| 'repaired'
	| 'written_off'

const EQUIPMENT_EVENT_LABELS: Record<EquipmentEventType, string> = {
	created: 'Создано',
	updated: 'Изменено',
	assigned: 'Выдано',
	unassigned: 'Возвращено',
	moved: 'Перемещено',
	status_changed: 'Смена статуса',
	component_added: 'Добавлен компонент',
	component_removed: 'Удалён компонент',
	component_replaced: 'Замена компонента',
	repaired: 'Ремонт',
	written_off: 'Списание',
}

export const getEquipmentEventLabel = (type: EquipmentEventType) => EQUIPMENT_EVENT_LABELS[type] ?? type

export interface IEquipmentEvent {
	id: string
	equipmentId: string
	eventType: EquipmentEventType
	operationId: string
	componentId: string | null
	oldValue: Record<string, unknown> | null
	newValue: Record<string, unknown> | null
	userId: string | null
	comment: string
	createdAt: string
}

export interface IKitComposition {
	id: string
	kitId: string
	componentId: string
	startedAt: string
	endedAt: string | null
	startedBy: string | null
	endedBy: string | null
	reason: string
	actId: string | null
}

export interface IAssignEquipmentDTO {
	userId?: string | null
	holderName?: string | null
	siteId?: string | null
	location?: string | null
	comment?: string
	force?: boolean
	skipAct?: boolean
}

export interface IUnassignEquipmentDTO {
	siteId?: string | null
	location?: string | null
	comment?: string
}

export interface IChangeStatusDTO {
	status: EquipmentStatus
	comment?: string
}

export interface ICompositionOpDTO {
	components: { componentId: string }[]
	reason?: string
}

export interface IActPosition {
	equipmentId: string
	typeName: string
	name: string
	serialNumber: string | null
	inventoryNumber: string | null
}

export interface IEquipmentAct {
	id: string
	realmId: string
	number: string
	kitEquipmentId: string | null
	recipientUserId: string | null
	recipientName: string | null
	siteId: string | null
	location: string | null
	issuedBy: string | null
	issuedAt: string
	content: IActPosition[]
	comment: string
	createdAt: string
}

export interface IEquipmentActDTO {
	kitEquipmentId?: string | null
	recipientUserId?: string | null
	recipientName?: string | null
	siteId?: string | null
	location?: string | null
	comment?: string
}

export interface IEquipmentActFilter {
	kitId?: string
	limit?: number
	offset?: number
}

export interface IEquipmentActList {
	data: IEquipmentAct[]
	total: number
}