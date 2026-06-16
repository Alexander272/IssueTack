import type { IRolePermissionItem } from '../../types/role'

const cyrillicToLatin: Record<string, string> = {
	а: 'a',
	б: 'b',
	в: 'v',
	г: 'g',
	д: 'd',
	е: 'e',
	ё: 'yo',
	ж: 'zh',
	з: 'z',
	и: 'i',
	й: 'y',
	к: 'k',
	л: 'l',
	м: 'm',
	н: 'n',
	о: 'o',
	п: 'p',
	р: 'r',
	с: 's',
	т: 't',
	у: 'u',
	ф: 'f',
	х: 'h',
	ц: 'ts',
	ч: 'ch',
	ш: 'sh',
	щ: 'sch',
	ъ: '',
	ы: 'y',
	ь: '',
	э: 'e',
	ю: 'yu',
	я: 'ya',
}

export const transliterate = (text: string): string => {
	return text
		.toLowerCase()
		.split('')
		.map(char => cyrillicToLatin[char] || char)
		.join('')
}

export const generateSlug = (name: string) => {
	return transliterate(name)
		.replace(/[^a-z0-9\s-]/gi, '')
		.replace(/\s+/g, '-')
		.replace(/-+/g, '-')
		.replace(/^-|-$/g, '')
		.substring(0, 50)
}

export const formatAction = (action: string) => {
	switch (action) {
		case 'read':
			return 'Чтение'
		case 'write':
			return 'Создание/Редактирование'
		case 'delete':
			return 'Удаление'
		case '*':
			return 'Все'

		default:
			return action
	}
}

export const actionSort = (a: IRolePermissionItem, b: IRolePermissionItem) => {
	const aPriority = ['read', 'write', 'delete', '*']
	const bPriority = ['read', 'write', 'delete', '*']
	return aPriority.indexOf(a.action) - bPriority.indexOf(b.action)
}

const BADGE_STYLES = {
	active: { bg: '#d1f5dd', color: '#22c55e', label: 'Включено', text: '✓' },
	activeChanged: { bg: '#fff3cd', color: '#2e8f51', label: 'Включено (Изменено)', text: '✓' },
	empty: { bg: '#ffd1d9', color: '#9c5353', label: 'Выключено', text: '✕' },
	emptyChanged: { bg: '#fff3cd', color: '#f59e0b', label: 'Выключено (Изменено)', text: '✕' },
	inherited: { bg: '#71b8ff', color: '#355583', label: 'Унаследовано', text: '↳' },
	disabled: { bg: '#e5e7eb', color: '#9ca3af', label: 'Недоступно', text: '–' },
}

export const getBadge = (allowed: boolean, item?: IRolePermissionItem) => {
	if (!allowed) return BADGE_STYLES.disabled
	if (item?.isInherited) return BADGE_STYLES.inherited
	if (item?.isAssigned) {
		if (item.status === 'changed') return BADGE_STYLES.activeChanged
		return BADGE_STYLES.active
	}
	if (item?.status === 'changed') return BADGE_STYLES.emptyChanged
	return BADGE_STYLES.empty
}
