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