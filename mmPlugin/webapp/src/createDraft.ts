// Черновик формы создания заявки в плагине. Хранится в sessionStorage (не
// переживает закрытие вкладки браузера): модалка размонтируется при закрытии и
// при переключении вкладок, а стейт React вместе с ней. Ключ включает realm и
// пользователя, чтобы на общем компьютере не подхватить чужой черновик.
// Вложения (File) не сохраняются — их нельзя сериализовать.

export interface PluginCreateDraft {
	title: string
	description: string
	categoryId: string
	siteId: string
	// Доп. секции менеджера/исполнителя — опциональны: поле появляется только
	// если пользователь выбрал его, у «чистого» заявителя их нет вовсе.
	priority?: string
	groupId?: string
	assigneeId?: string
	ownerId?: string
	dueDate?: string
}

const VERSION = 2

function draftKey(realmId: string, userId: string) {
	return `it-plugin-create-draft:${realmId}:${userId}`
}

// isOptionalStr — строка либо отсутствует вовсе (не «»): выбор «не выбрано»
// в селекте тоже валиден как пропущенное значение.
function isOptionalStr(v: unknown): boolean {
	return v === undefined || typeof v === 'string'
}

function isDraft(value: unknown): value is PluginCreateDraft {
	if (!value || typeof value !== 'object') return false
	const d = value as Record<string, unknown>
	return (
		typeof d.title === 'string' &&
		typeof d.description === 'string' &&
		typeof d.categoryId === 'string' &&
		typeof d.siteId === 'string' &&
		isOptionalStr(d.priority) &&
		isOptionalStr(d.groupId) &&
		isOptionalStr(d.assigneeId) &&
		isOptionalStr(d.ownerId) &&
		isOptionalStr(d.dueDate)
	)
}

export function loadDraft(realmId: string, userId: string): PluginCreateDraft | null {
	try {
		const raw = window.sessionStorage.getItem(draftKey(realmId, userId))
		if (!raw) return null
		const parsed = JSON.parse(raw) as { v?: unknown; draft?: unknown }
		if (parsed?.v !== VERSION || !isDraft(parsed.draft)) {
			clearDraft(realmId, userId)
			return null
		}
		return parsed.draft
	} catch {
		clearDraft(realmId, userId)
		return null
	}
}

export function saveDraft(realmId: string, userId: string, draft: PluginCreateDraft) {
	try {
		window.sessionStorage.setItem(draftKey(realmId, userId), JSON.stringify({ v: VERSION, draft }))
	} catch {
		// Хранилище недоступно/переполнено — черновик просто не сохранится.
	}
}

export function clearDraft(realmId: string, userId: string) {
	try {
		window.sessionStorage.removeItem(draftKey(realmId, userId))
	} catch {
		// ignore
	}
}