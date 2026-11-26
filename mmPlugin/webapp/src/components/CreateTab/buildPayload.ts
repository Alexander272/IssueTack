// Сборка тела создания заявки из значений формы. Поля доп. секций отправляются
// только тем, кому они доступны: менеджер — все (приоритет, группа, исполнитель,
// заявитель, срок), исполнитель — заказчик и самоназначение. Обычный заявитель
// может указать только наблюдателей. Группа/приоритет при пустых полях сервер
// берёт из категории.

import type {CreateTicketPayload} from '../../api'
import type {PluginCalendar} from '../../types'
import {dueDateValueForSubmit} from '../../workDays'
import type {AdvancedValues} from './model'

interface BuildPayloadParams {
	isManager: boolean
	isExecutor: boolean
	canPickModifier: boolean
	title: string
	description: string
	categoryId: string
	categoryModifierId: string
	siteId: string
	files: File[]
	advanced: AdvancedValues
	calendar?: PluginCalendar
}

export function buildCreatePayload({
	isManager,
	isExecutor,
	canPickModifier,
	title,
	description,
	categoryId,
	categoryModifierId,
	siteId,
	files,
	advanced,
	calendar,
}: BuildPayloadParams): CreateTicketPayload {
	const payload: CreateTicketPayload = {
		title: title.trim(),
		description: description.trim(),
		categoryId: categoryId || null,
		...(canPickModifier && categoryModifierId ? {categoryModifierId} : {}),
		siteId: siteId || null,
		files,
	}
	const {priority, ownerId, ownerMuted, groupId, assigneeId, dueDate, notifierIds} = advanced
	if (isManager) {
		payload.priority = priority
		if (groupId) payload.groupId = groupId
		if (assigneeId) payload.assigneeId = assigneeId
		if (ownerId) payload.ownerId = ownerId
		if (dueDate) payload.dueDate = dueDateValueForSubmit(dueDate, calendar)
		if (ownerMuted && ownerId) payload.ownerMuted = true
		payload.notifierIds = notifierIds
	} else if (isExecutor) {
		if (ownerId) payload.ownerId = ownerId
		if (assigneeId) payload.assigneeId = assigneeId
		if (priority) payload.priority = priority
		if (dueDate) payload.dueDate = dueDateValueForSubmit(dueDate, calendar)
		payload.notifierIds = notifierIds
	} else if (notifierIds.length > 0) {
		payload.notifierIds = notifierIds
	}
	return payload
}
