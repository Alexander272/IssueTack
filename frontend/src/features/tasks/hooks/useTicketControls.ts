import { useMemo } from 'react'

import type { ITask } from '../types/task'
import { INACTIVE_STATUSES } from '../constants/taskMaps'

export interface TicketControls {
	/** Заявка в терминальном статусе: данные и решения недоступны. */
	isInactive: boolean
	/** Правка заголовка/описания и полей: canEditFields или админ области (не на замороженной). */
	canEdit: boolean
	/** «Рабочий» доступ: write-права по атрибутам или исполнитель. */
	canWork: boolean
	/** Создание подзадач — уже учитывает заморозку (canCreateSubtask с бэкенда). */
	canCreateSubtask: boolean
	/** Вложение файла — уже учитывает заморозку (canAttachToTicket с бэкенда). */
	canAttachToTicket: boolean
	/** Управление тикетом: начальник области или менеджер его группы. */
	canManage: boolean
	/** Удаление подзадачи: право удаления тикета + work-доступ, только на активной заявке. */
	canDeleteSubtask: boolean
}

/**
 * useTicketControls — единая точка прав для контролов детали заявки. Раньше
 * TaskDetail собирал их из access-флагов, capabilities и статуса, а компоненты
 * получали разрозненные пропы; теперь формула живёт здесь, а компоненты читают
 * готовые значения.
 */
export const useTicketControls = (task: ITask | undefined): TicketControls =>
	useMemo(() => {
		const isInactive = task ? INACTIVE_STATUSES.includes(task.status) : false
		const access = task?.access

		return {
			isInactive,
			canEdit: !isInactive && Boolean(access?.canEditFields || access?.isAdmin),
			canWork: Boolean(access?.canWork),
			canCreateSubtask: Boolean(access?.canCreateSubtask),
			canAttachToTicket: Boolean(access?.canAttachToTicket),
			canManage: Boolean(access?.canManage),
			canDeleteSubtask: Boolean(access?.canDelete) && Boolean(access?.canWork) && !isInactive,
		}
	}, [task])
