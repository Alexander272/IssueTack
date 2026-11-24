import type { IUserShort } from '@/features/user/types/user'
import type { IComment } from './comment'

export type TicketStatus = 'open' | 'in_progress' | 'pending' | 'on_hold' | 'resolved' | 'closed' | 'cancelled'

export type Priority = 'low' | 'medium' | 'high' | 'urgent'

export interface ISiteShort {
	id: string
	name: string
}

export interface ICategoryShort {
	id: string
	name: string
	categoryGroupName?: string | null
}

export interface ICategoryModifierShort {
	id: string
	name: string
}

export interface IGroupShort {
	id: string
	name: string
}

export interface IEquipmentShort {
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
	commentId?: string
	createdAt: string
}

export interface IAccessFlags {
	canRead: boolean
	canWrite: boolean
	canDelete: boolean
	canWork: boolean
	canTake: boolean
	canEditFields: boolean
	isAdmin: boolean
	isManager: boolean
	/** Решения владельца: отменить новую заявку (open → cancelled). */
	canCancel: boolean
	/** Решения владельца: подтвердить решение (resolved → closed). */
	canConfirm: boolean
	/** Решения владельца: вернуть в работу (resolved → in_progress). */
	canReopen: boolean
	allowedStatuses: TicketStatus[]
	/** Могут ли текущий пользователь-исполнитель запросить перенос срока (при установленном сроке). */
	canRequestDueDate?: boolean
	/** Могут ли текущий пользователь (менеджер группы/начальник области) утверждать или отклонять запрос переноса срока. */
	canDecideDueDate?: boolean
	/** Создание подзадачи: создатель/исполнитель/управляющий тикетом (уже учитывает заморозку). */
	canCreateSubtask?: boolean
	/** Загрузка вложения: work-доступ или заказчик активной заявки (уже учитывает заморозку). */
	canAttachToTicket?: boolean
	/** Управление тикетом: начальник области или менеджер его группы. */
	canManage?: boolean
	/** Выбор подкатегории: исполнитель или начальник области, пока заявка открыта. */
	canPickModifier?: boolean
	/** Отметить заявку дублем другой: рабочий доступ или управляющий тикетом. */
	canLinkDuplicate?: boolean
	/** Отвязать дубль: тот же доступ либо автор дубля. */
	canUnlinkDuplicate?: boolean
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
	categoryModifier?: ICategoryModifierShort | null
	creator: IUserShort
	owner: IUserShort | null
	group: IGroupShort | null
	assignee: IUserShort | null
	manager: IUserShort | null
	dueDate: string | null
	closedAt: string | null
	resolvedAt: string | null
	createdAt: string
	updatedAt: string
	subtasks?: ISubtask[]
	attachments?: IAttachment[]
	comments?: IComment[]
	notifiers?: IUserShort[]
	access?: IAccessFlags
	/** Отключает уведомления заказчику и скрывает заявку из его списка заявок. */
	ownerMuted?: boolean
	/** Запрос исполнителя на перенос срока (только когда есть pending-решение менеджера). */
	dueDateRequest?: IDueDateRequest | null
	equipment?: IEquipmentShort | null
	/** Основная заявка, для которой эта помечена дублем (дубли всегда закрыты). */
	duplicateOf?: string | null
	/** Номер основной заявки — для бейджа «дубль #N» без лишнего запроса. */
	duplicateOfNumber?: number | null
	/** Статус дубля на момент привязки; возвращается при отвязке. */
	previousStatus?: TicketStatus | null
}

// Запрос исполнителя на перенос срока заявки (статус pending → approved/rejected).
export interface IDueDateRequest {
	id: string
	ticketId: string
	requesterId: string
	requester?: IUserShort | null
	currentDueDate?: string | null
	proposedDueDate: string
	reason: string
	status: 'pending' | 'approved' | 'rejected'
	decidedBy?: string | null
	decidedAt?: string | null
	decisionComment?: string | null
	createdAt: string
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
	archived?: boolean
	/**
	 * Отдать заявки всех статусов, минуя вкладку «Активные/Архив» (statusMode): история
	 * единицы оборудования включает закрытые/отменённые. Заполняется deep-link'ом
	 * `?equipmentId=` на странице заявок и блоком «Заявки по оборудованию».
	 */
	allStatuses?: boolean
	favoritesType?: 'temporary' | 'permanent'
	limit?: number
	offset?: number
	mode?: string
	equipmentId?: string
	/** true — только дубли, false — только не-дубли. */
	duplicates?: boolean
}

/** Ответ привязки/отвязки дубля: корень цепочки и путь от дубля к корню. */
export interface IDuplicateLinkResult {
	rootId: string
	chain: string[]
}

/** Карточка связанной заявки на основной. Статус намеренно не отдаётся: дубли всегда закрыты. */
export interface IDuplicateRef {
	id: string
	ticketNumber?: number
	title: string
	creatorName?: string
	assigneeName?: string
	commentsCount: number
	attachmentsCount: number
	createdAt: string
	/** Право отвязать именно этого дубля — считается бэкендом для каждой строки. */
	canUnlink: boolean
}

export interface ISubtaskCreateDTO {
	title: string
	description?: string
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
	/** Подкатегория: разово перезаписывает срок и приоритет новой заявки. */
	categoryModifierId?: string | null
	creatorId: string
	ownerId?: string | null
	groupId?: string | null
	assigneeId?: string | null
	managerId?: string | null
	dueDate?: string | null
	closedAt?: string | null
	resolvedAt?: string | null
	/** Подзадачи, создаваемые вместе с заявкой (автор, статус, приоритет и порядок проставит бэкенд). */
	subtasks?: ISubtaskCreateDTO[]
	/** Наблюдатели заявки (список «Уведомлять»). При обновлении — полная замена, если передано. */
	notifierIds?: string[]
	/** Отключает уведомления заказчику и скрывает заявку из его списка заявок. */
	ownerMuted?: boolean
	equipmentId?: string | null
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
	/** Автор подзадачи: правка содержимого доступна автору, менеджеру группы или админу реалма. */
	createdBy?: string | null
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
