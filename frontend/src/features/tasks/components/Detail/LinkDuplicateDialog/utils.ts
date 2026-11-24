import type { ITask } from '../../../types/task'

export const ticketNumber = (task: ITask) => task.ticketNumber ?? '—'

export const personName = (user: ITask['creator'] | null) => {
	if (!user) return '—'
	return `${user.lastName} ${user.firstName}`.trim() || user.username
}

/** «№123», «No 123», «#123» → «123». Префикс срезаем только перед цифрами, текстовые запросы не трогаем. */
export const normalizeQuery = (raw: string) => raw.trim().replace(/^(?:№|No|#)\s*(?=\d)/i, '')

/** Причина, по которой заявку нельзя выбрать в качестве основной. */
export const unavailableReason = (candidate: ITask, task: ITask): string | null => {
	if (candidate.id === task.id) return 'Это текущая заявка'
	if (candidate.duplicateOf === task.id) return 'Уже является дублем этой заявки'
	// Закрытая и решённая основная допустимы: дубль закрывается своей операцией,
	// а при unlink ему возвращается сохранённый previous_status.
	if (candidate.status === 'cancelled') return 'Отменена: привязать к ней нельзя'
	return null
}

export const linkSuccessMessage = (task: ITask, picked: ITask): string =>
	picked.duplicateOf
		? `№${ticketNumber(task)} привязана к основной №${picked.duplicateOfNumber ?? '—'} (через №${ticketNumber(picked)})`
		: `№${ticketNumber(task)} отмечена дублем №${ticketNumber(picked)}`
