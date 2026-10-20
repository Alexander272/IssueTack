export type EquipmentTypeKind = 'device' | 'component' | 'periphery' | 'consumable'

export const EQUIPMENT_TYPE_KIND_LABELS: Record<EquipmentTypeKind, string> = {
	device: 'Устройство',
	component: 'Комплектующее',
	periphery: 'Периферия',
	consumable: 'Расходник',
}

export const EQUIPMENT_TYPE_KIND_OPTIONS = (Object.keys(EQUIPMENT_TYPE_KIND_LABELS) as EquipmentTypeKind[]).map(value => ({
	value,
	label: EQUIPMENT_TYPE_KIND_LABELS[value],
}))

// canBePart сообщает, можно ли единицу такого типа включить в физический состав
// (parent_id): комплектующее или расходник.
export const canBePart = (kind?: EquipmentTypeKind) => kind === 'component' || kind === 'consumable'

export interface IEquipmentType {
	id: string
	realmId: string
	name: string
	kind: EquipmentTypeKind
	// icon — ключ из реестра typeIcons; '' = «Авто» (подбор по названию типа).
	icon: string
	sortOrder: number
	createdAt: string
	updatedAt: string
}

export interface IEquipmentTypeDTO {
	id?: string
	realmId?: string
	name: string
	kind: EquipmentTypeKind
	icon: string
	sortOrder: number
}

export type EquipmentStatus =
	| 'in_stock'
	| 'in_use'
	| 'repair'
	| 'defective'
	| 'reserved'
	| 'written_off'

const EQUIPMENT_STATUS_MAP: Record<EquipmentStatus, { label: string; color: string }> = {
	in_stock: { label: 'На складе', color: '#6b7280' },
	in_use: { label: 'В эксплуатации', color: '#16a34a' },
	repair: { label: 'В ремонте', color: '#ea580c' },
	defective: { label: 'Дефектная', color: '#ca8a04' },
	reserved: { label: 'Зарезервировано', color: '#2563eb' },
	written_off: { label: 'Списано', color: '#dc2626' },
}

export const getEquipmentStatus = (status: EquipmentStatus) =>
	EQUIPMENT_STATUS_MAP[status] ?? { label: status, color: '#6b7280' }

export const EQUIPMENT_STATUS_OPTIONS = Object.entries(EQUIPMENT_STATUS_MAP).map(([value, { label }]) => ({
	value: value as EquipmentStatus,
	label,
}))

// Статусы, которые выводятся из держателя, а не выбираются вручную: «в
// эксплуатации» ставит выдача, «на складе» — возврат (бэкенд делает это в
// Assign/Unassign и повторно в Create/Update). Поэтому их нет в селекте статуса
// формы, а в шапке карточки они не дублируются чипом — про «на складе»/«в
// эксплуатации» и так говорит блок «Держатель» ниже.
export const HOLDER_DERIVED_STATUSES: EquipmentStatus[] = ['in_stock', 'in_use']

export interface IEquipment {
	id: string
	realmId: string
	// typeId — null у комплекта: у комплекта нет типа (только у единиц).
	typeId: string | null
	typeName: string
	typeKind: EquipmentTypeKind
	// typeIcon — явный ключ иконки типа; '' = «Авто» (подбор по typeName).
	typeIcon: string
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
	// returnDueAt — срок возврата текущей (открытой) выдачи. null — держателя нет
	// или срок не задавали. По нему карточка рисует «просрочен возврат».
	returnDueAt: string | null
	location: string | null
	purchaseDate: string | null
	warrantyUntil: string | null
	notes: string
	// containerName — имя контейнера для подписи «внутри …» в плоском списке
	// (физический родитель или логический комплект).
	containerName: string | null
	isKit: boolean
	kitId: string | null
	createdBy: string | null
	createdAt: string
	updatedAt: string
}

export interface IEquipmentDTO {
	id?: string
	typeId?: string
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
	notes?: string
	isKit?: boolean
	kitId?: string | null
	// components — стартовый состав комплекта, принимается только при создании
	// (POST /equipment). В PUT поле игнорируется: состав правится отдельными
	// операциями состава.
	components?: { componentId: string }[]
}

export interface IEquipmentFilter {
	typeId?: string
	status?: EquipmentStatus
	isKit?: boolean
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

// IEquipmentStats — счётчики шапки страницы: единицы, комплекты и
// комплектующие/расходники на складе.
export interface IEquipmentStats {
	units: number
	kits: number
	stockParts: number
}

// IEquipmentTreeNode — узел дерева оборудования: единица с текущими детьми
// (члены логического комплекта и/или физический состав). Дерево грузится лениво,
// поэтому children заполнены только у раскрытого уровня, а hasChildren/childCount
// приходят с сервера — по ним рисуется шеврон и счётчик «(N)» без загрузки детей.
export interface IEquipmentTreeNode extends IEquipment {
	children: IEquipmentTreeNode[]
	hasChildren: boolean
	childCount: number
}

export interface IEquipmentTreeList {
	data: IEquipmentTreeNode[]
	total: number
}

// IEquipmentTreeArgs — параметры одного уровня дерева: parentId = undefined — корни,
// иначе дети узла. status/isKit применяются только к корням (как было раньше на клиенте).
export interface IEquipmentTreeArgs {
	parentId?: string
	status?: EquipmentStatus
	isKit?: boolean
	limit?: number
	offset?: number
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
	// returnDueAt — ожидаемый срок возврата (null = срок не задавали). Реквизит
	// конкретной выдачи.
	returnDueAt: string | null
	assignedBy: string | null
	comment: string
	// Подписи ссылок резолвит бэкенд (SQL-джойн): клиенту не нужен справочник
	// пользователей и площадок ради строки истории.
	holderFullName: string
	siteName: string
	assignedByName: string
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
	// ФИО актора резолвит бэкенд (SQL-джойн по user_id).
	userName: string
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
	// Подписи ссылок резолвит бэкенд (SQL-джойн).
	componentName: string
	startedByName: string
	endedByName: string
}

export interface IAssignEquipmentDTO {
	userId?: string | null
	holderName?: string | null
	siteId?: string | null
	location?: string | null
	comment?: string
	force?: boolean
	skipAct?: boolean
	// assignedAt — момент выдачи (RFC3339), его отдаёт DateTimePicker; не
	// передан — сервер поставит текущее время. Будущее отклоняется (EQ017).
	assignedAt?: string
	// returnDueAt — ожидаемый срок возврата (RFC3339). Не передан — выдают
	// бессрочно. Раньше даты выдачи быть не может (EQ019).
	returnDueAt?: string | null
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
	// status: 'formed' — docx сформирован и выдан пользователю, 'signed' —
	// подгружен скан/фото подписанного акта.
	status: EquipmentActStatus
	kitEquipmentId: string | null
	recipientUserId: string | null
	recipientName: string | null
	siteId: string | null
	location: string | null
	issuedBy: string | null
	issuedAt: string
	content: IActPosition[]
	comment: string
	signedFileName: string | null
	signedMimeType: string | null
	signedFileSize: number | null
	signedAt: string | null
	signedBy: string | null
	createdAt: string
}

export type EquipmentActStatus = 'formed' | 'signed'

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
	// equipmentId — акты единицы: где она комплект и где позиция состава.
	equipmentId?: string
	limit?: number
	offset?: number
}

export interface IEquipmentActList {
	data: IEquipmentAct[]
	total: number
}

// ===== Шаблоны комплектов =====

// IEquipmentKitTemplate — именованный набор типов оборудования, из которого
// собирают комплект («Стандартное рабочее место»). Позиция шаблона — именно ТИП:
// при применении шаблона слоты заполняются типами, а конкретные единицы
// выбираются со склада.
export interface IEquipmentKitTemplate {
	id: string
	realmId: string
	name: string
	description: string
	sortOrder: number
	items: IEquipmentKitTemplateItem[]
	createdAt: string
	updatedAt: string
}

export interface IEquipmentKitTemplateItem {
	id: string
	templateId: string
	typeId: string
	// typeName/typeKind резолвит сервер джойном, чтобы не тянуть справочник типов.
	typeName: string
	typeKind: EquipmentTypeKind
	sortOrder: number
}

export interface IEquipmentKitTemplateItemDTO {
	typeId: string
	sortOrder?: number
}

export interface IEquipmentKitTemplateDTO {
	id?: string
	name: string
	description?: string
	sortOrder?: number
	items: IEquipmentKitTemplateItemDTO[]
}