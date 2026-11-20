import { createContext, useContext } from 'react'

/**
 * Режим формы заявки. От него зависит поведение полей: на создании категория
 * подставляет группу/приоритет/срок (autoAssign), на редактировании — нет.
 */
export type TaskFormMode = 'create' | 'edit'

/**
 * TaskFormSettings — права и режим, общие для секций формы создания/редактирования.
 *
 * Раньше TaskCreateForm и TaskEditForm прокидывали в AdvancedSettingsSection и
 * CustomerAndDateSection по 6–8 однотипных пропов, причём значения дублировали
 * друг друга и легко расходились. Теперь набор считается один раз в форме и
 * лежит в контексте, а секции читают только то, что им нужно.
 */
export interface TaskFormSettings {
	mode: TaskFormMode
	/** Автор заявки — исключается из списка «Уведомлять». */
	creatorId: string | null
	/** Можно править заголовок/описание (canEditFields; не включает админа области). */
	canEditContent: boolean
	/** Можно менять срок: менеджер группы/админ области. */
	canEditDueDate: boolean
	/** Срок дополнительно закрыт заявкой с подкатегорией (правят только избранные роли). */
	dueDateDisabled: boolean
	/**
	 * Дата создания заявки — от неё подкатегория считает срок. На создании её нет
	 * (отсчёт идёт от сейчас), в форме редактирования берётся task.createdAt.
	 */
	dueDateBase?: string
	/**
	 * Можно выбирать подкатегорию: исполнитель (участник группы) или начальник
	 * области. При создании право не зависит от выбранного исполнителя заявки.
	 */
	canPickModifier: boolean
	/** Можно выбирать исполнителя. */
	canPickAssignee: boolean
	/** Можно менять приоритет. */
	canEditPriority: boolean
	/** Можно менять группу. */
	canEditGroup: boolean
	/** Можно отключить уведомления заказчику и скрыть заявку из его списка. */
	canEditOwnerMuted: boolean
	/** Доступен список «Уведомлять» (скрыт у безгруппового пользователя). */
	canEditNotifiers: boolean
	/** Показывать поле «Заказчик» в CustomerAndDateSection. */
	showOwner: boolean
	/** Показывать поле «Срок выполнения» в CustomerAndDateSection. */
	showDueDate: boolean
	/** Показывать подпись поля «Уведомлять» (заголовок секции может её заменять). */
	showNotifierFieldLabel: boolean
}

export const TaskFormSettingsContext = createContext<TaskFormSettings | null>(null)

export const useTaskFormSettings = (): TaskFormSettings => {
	const settings = useContext(TaskFormSettingsContext)
	if (!settings) {
		throw new Error('useTaskFormSettings must be used inside TaskFormProvider')
	}
	return settings
}

export const useTaskFormMode = (): TaskFormMode => useTaskFormSettings().mode
