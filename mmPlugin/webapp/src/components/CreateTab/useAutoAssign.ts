// Авто-назначение в стиле веба (AdvancedSettingsSection + onChange категории
// в TaskCreateForm): смена категории подставляет группу и дефолтный приоритет
// категории, подкатегории — приоритет/срок из подкатегории, смена группы —
// дефолтного исполнителя. Срабатывает только на реальном изменении поля
// (useRef-пред), а не на первом рендере или восстановлении черновика — иначе
// авто-подстановка затирала бы сохранённые значения. Сервер всё равно авторитетен:
// подставленные здесь значения совпадают с теми, что TicketService.Create
// применил бы сам при пустых полях.

import {useEffect, useRef} from 'react'

import type {PluginCalendar, PluginCategory, PluginGroupShort} from '../../types'
import {autoDueDateLocal} from '../../workDays'

interface AutoAssignParams {
	categoryId: string
	categoryModifierId: string
	groupId: string
	categories: PluginCategory[]
	groups: PluginGroupShort[]
	calendar?: PluginCalendar
	isManager: boolean
	isExecutor: boolean
	onModifierReset: () => void
	setCategoryModifierId: (id: string) => void
	setGroupId: (id: string) => void
	setPriority: (priority: string) => void
	setDueDate: (dueDate: string) => void
	setAssigneeId: (id: string) => void
}

export function useAutoAssign({
	categoryId,
	categoryModifierId,
	groupId,
	categories,
	groups,
	calendar,
	isManager,
	isExecutor,
	onModifierReset,
	setCategoryModifierId,
	setGroupId,
	setPriority,
	setDueDate,
	setAssigneeId,
}: AutoAssignParams) {
	const prevCategoryId = useRef(categoryId)
	const prevGroupId = useRef(groupId)
	const prevModifierCategoryId = useRef(categoryId)
	const prevCategoryModifierId = useRef(categoryModifierId)

	// Подкатегория принадлежит категории: при смене категории форма сбрасывает её
	// следом за сервером (иначе в запрос ушёл бы чужой id).
	useEffect(() => {
		if (prevModifierCategoryId.current === categoryId) return
		prevModifierCategoryId.current = categoryId
		setCategoryModifierId('')
		onModifierReset()
	}, [categoryId, setCategoryModifierId, onModifierReset])

	// Категория → группа + приоритет + срок (менеджер И исполнитель).
	useEffect(() => {
		if (!(isManager || isExecutor)) return
		if (prevCategoryId.current === categoryId) return
		prevCategoryId.current = categoryId
		const cat = categories.find(c => c.id === categoryId)
		setGroupId(cat?.groupId ?? '')
		if (cat?.priority) setPriority(cat.priority)
		// Автосрок из категории — зеркало веба/бэкенда: дефолт категории
		// проставляется в поле срока в рабочих днях (с учётом выходных и праздников,
		// конец целевого рабочего дня в календарной TZ), а категория без автосрока
		// его сбрасывает, иначе в форме остался бы срок предыдущей категории.
		setDueDate(autoDueDateLocal(cat?.defaultDueDays, calendar) ?? '')
	}, [categoryId, categories, isManager, isExecutor, calendar, setGroupId, setPriority, setDueDate])

	// Подкатегория → приоритет + срок. Пустые поля наследуются от категории.
	useEffect(() => {
		if (prevCategoryModifierId.current === categoryModifierId) return
		prevCategoryModifierId.current = categoryModifierId
		if (!categoryModifierId) return
		const modifier = (categories.find(c => c.id === categoryId)?.modifiers || []).find(
			m => m.id === categoryModifierId,
		)
		if (!modifier) return
		const cat = categories.find(c => c.id === categoryId)
		setPriority(modifier.priority || cat?.priority || '')
		const days = modifier.defaultDueDays ?? cat?.defaultDueDays
		setDueDate(days != null && days > 0 ? (autoDueDateLocal(days, calendar) ?? '') : '')
	}, [categoryModifierId, categories, categoryId, calendar, setPriority, setDueDate])

	useEffect(() => {
		if (!isManager) return
		if (prevGroupId.current === groupId) return
		prevGroupId.current = groupId
		if (!groupId) {
			setAssigneeId('')
			return
		}
		const group = groups.find(g => g.id === groupId)
		setAssigneeId(group?.defaultAssigneeId ?? '')
	}, [groupId, groups, isManager, setAssigneeId])
}
