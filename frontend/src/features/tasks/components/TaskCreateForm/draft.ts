import type { FormValues } from './types'

// Черновик формы создания заявки. sessionStorage: не переживает закрытие
// вкладки браузера, но восстанавливает форму после закрытия модалки/перезагрузки
// страницы. Ключ включает realm и пользователя, чтобы на общем компьютере не
// подхватить чужой черновик. Вложения (File) не сохраняются — их нельзя
// сериализовать.

const VERSION = 5

function draftKey(realmId: string, userId: string) {
	return `@issueTrack/taskCreateDraft:${realmId}:${userId}`
}

function isSubtask(value: unknown): boolean {
	if (!value || typeof value !== 'object') return false
	const s = value as Record<string, unknown>
	return typeof s.title === 'string' && typeof s.description === 'string'
}

const isString = (val: unknown): val is string => typeof val === 'string'
const isNullableString = (val: unknown): val is string | null => val === null || typeof val === 'string'
const isStringArray = (val: unknown): val is string[] => Array.isArray(val) && val.every(isString)

function isFormValues(value: unknown): value is FormValues {
	if (!value || typeof value !== 'object') return false
	const v = value as Record<string, unknown>

	const requiredStrings = [v.title, v.description, v.priority, v.categoryId, v.siteId]
	const nullableStrings = [v.categoryModifierId, v.groupId, v.ownerId, v.assigneeId, v.dueDate, v.equipmentId]

	return (
		requiredStrings.every(isString) &&
		nullableStrings.every(isNullableString) &&
		typeof v.ownerMuted === 'boolean' &&
		isStringArray(v.notifierIds) &&
		Array.isArray(v.subtasks) &&
		v.subtasks.every(isSubtask)
	)
}

export function loadDraft(realmId: string, userId: string): FormValues | null {
	try {
		const raw = window.sessionStorage.getItem(draftKey(realmId, userId))
		if (!raw) return null
		const parsed = JSON.parse(raw) as { v?: unknown; data?: unknown }
		if (parsed?.v !== VERSION || !isFormValues(parsed.data)) {
			clearDraft(realmId, userId)
			return null
		}
		return parsed.data
	} catch {
		clearDraft(realmId, userId)
		return null
	}
}

export function saveDraft(realmId: string, userId: string, data: FormValues) {
	try {
		window.sessionStorage.setItem(draftKey(realmId, userId), JSON.stringify({ v: VERSION, data }))
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

// hasDraftContent — есть ли в форме то, что жалко потерять. Дефолтная площадка
// пользователя не считается черновиком (её проставляет эффект, а не человек).
export function hasDraftContent(values: FormValues, filesCount: number): boolean {
	if (filesCount > 0) return true
	if (values.ownerMuted) return true
	if (values.notifierIds.length > 0) return true

	const textFields = [values.title, values.description]
	if (textFields.some(text => text.trim().length > 0)) return true

	const selectedFields = [
		values.categoryId,
		values.categoryModifierId,
		values.groupId,
		values.ownerId,
		values.assigneeId,
		values.dueDate,
		values.equipmentId,
	]
	if (selectedFields.some(Boolean)) return true

	return values.subtasks.some(s => s.title.trim().length > 0 || s.description.trim().length > 0)
}
