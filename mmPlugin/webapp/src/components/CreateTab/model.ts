// Общие типы и константы формы создания заявки в плагине. Вынесены из CreateTab,
// чтобы секции формы и хуки не тянули друг друга и не дублировали подписи.

import type {PluginUserShort, Priority} from '../../types'

export interface FieldErrors {
	title?: string
	categoryId?: string
	siteId?: string
	ownerId?: string
	priority?: string
}

// Лимиты вложений совпадают с бэкенд-плагином (handlers/mattermost/plugin.go):
// 10 файлов и 64 МБ на один запрос создания.
export const MAX_SEED_FILES = 10
export const MAX_SEED_FILE_BYTES = 64 << 20

// PRIORITY_ORDER — порядок карточек приоритета, PRIORITY_LABELS/COLORS/DESCRIPTIONS —
// как в вебе (PRIORITY_MAP + priorityDescriptions в TaskCreateForm).
export const PRIORITY_ORDER: Priority[] = ['low', 'medium', 'high', 'urgent']
export const PRIORITY_LABELS: Record<Priority, string> = {
	low: 'Низкий',
	medium: 'Средний',
	high: 'Высокий',
	urgent: 'Критичный',
}
export const PRIORITY_COLORS: Record<Priority, string> = {
	low: '#10b981',
	medium: '#f59e0b',
	high: '#ef4444',
	urgent: '#dc2626',
}
export const PRIORITY_DESCRIPTIONS: Record<Priority, string> = {
	low: 'Не мешает работе, можно подождать',
	medium: 'Работа затруднена, есть обходной путь',
	high: 'Работа полностью остановлена',
	urgent: 'Критично, требуется немедленная реакция',
}

// pluginUserLabel — подпись пользователя в селектах формы: «Фамилия Имя
// (username)», без имён — просто username (как в диалогах бота).
export function pluginUserLabel(u: PluginUserShort): string {
	if (u.lastName || u.firstName) return `${u.lastName} ${u.firstName}`.trim() + ` (${u.username})`
	return u.username
}

// Поля секции 3, которые меняются вместе. Передаются в секцию одним объектом,
// чтобы число пропсов не раздувалось, а родитель обновлял только нужные поля.
export interface AdvancedValues {
	priority: string
	ownerId: string
	ownerMuted: boolean
	groupId: string
	assigneeId: string
	dueDate: string
	notifierIds: string[]
}
