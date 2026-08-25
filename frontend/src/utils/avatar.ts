const AVATAR_COLORS = [
	'#3b82f6',
	'#8b5cf6',
	'#ec4899',
	'#f59e0b',
	'#10b981',
	'#6366f1',
	'#14b8a6',
	'#f97316',
	'#a855f7',
	'#06b6d4',
	'#84cc16',
	'#0ea5e9',
	'#d946ef',
	'#eab308',
	'#22d3ee',
	'#7c3aed',
]

const hashCode = (str: string): number => {
	let hash = 0
	for (let i = 0; i < str.length; i++) {
		hash = str.charCodeAt(i) + ((hash << 5) - hash)
	}
	return hash
}

export function getAvatarColor(id: string): string {
	return AVATAR_COLORS[Math.abs(hashCode(id)) % AVATAR_COLORS.length]
}

type UserLike = { firstName?: string; lastName?: string; username?: string } | string | null | undefined

export function getInitials(user: UserLike): string {
	if (!user) return '?'
	if (typeof user === 'string') {
		const parts = user.trim().split(/\s+/)
		if (parts.length >= 2) return `${parts[0][0]}${parts[1][0]}`.toUpperCase()
		return user.slice(0, 2).toUpperCase()
	}
	if (user.firstName || user.lastName) {
		const first = user.firstName?.[0] ?? ''
		const last = user.lastName?.[0] ?? ''
		if (first || last) return `${first}${last}`.toUpperCase()
	}
	const name = user.username ?? '?'
	return name.slice(0, 2).toUpperCase()
}

export function getDisplayName(user?: UserLike): string {
	if (!user || typeof user === 'string') return user ?? 'Пользователь'
	if (user.firstName || user.lastName) {
		return `${user.firstName ?? ''} ${user.lastName ?? ''}`.trim()
	}
	return user.username ?? 'Пользователь'
}
