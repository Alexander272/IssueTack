import type { IUserData } from '@/features/user/types/user'

const avatarColors = ['#4f46e5', '#0891b2', '#059669', '#d97706', '#dc2626', '#7c3aed', '#db2777', '#2563eb']

export function getAvatarColor(id: string) {
	// Превращаем строку в число (суммируем коды символов)
	const hash = id.split('').reduce((acc, char) => {
		return char.charCodeAt(0) + ((acc << 5) - acc)
	}, 0)

	// Гарантируем положительное число и берем остаток от деления
	const index = Math.abs(hash) % avatarColors.length
	return avatarColors[index]
}

export function getInitials(u: IUserData) {
	return (u.firstName[0] || '') + (u.lastName[0] || '')
}
