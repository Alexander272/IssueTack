const CREDENTIALS_KEY = '@issueTrack/rememberedCredentials'
const CREDENTIALS_VERSION = 1

const OBFUSCATION_PEPPER = 'issueTrack/rememberedCredentials/v1'

export const CREDENTIALS_TTL_MS = 30 * 24 * 60 * 60 * 1000

export interface IRememberedCredentials {
	username: string
	password: string
}

type StoredCredentials = {
	v: number
	d: string
	c: string
	savedAt: number
}

type CredentialsPayload = {
	u: string
	p: string
}

const encoder = new TextEncoder()
const decoder = new TextDecoder()

const obfuscate = (value: string): string => {
	const pepper = encoder.encode(OBFUSCATION_PEPPER)
	const bytes = encoder.encode(value)
	const mixed = new Uint8Array(bytes.length)

	for (let i = 0; i < bytes.length; i++) {
		mixed[i] = bytes[i] ^ pepper[i % pepper.length]
	}

	let binary = ''
	for (const byte of mixed) binary += String.fromCharCode(byte)

	return btoa(binary)
}

const deobfuscate = (value: string): string => {
	const binary = atob(value)
	const mixed = new Uint8Array(binary.length)

	for (let i = 0; i < binary.length; i++) mixed[i] = binary.charCodeAt(i)

	const pepper = encoder.encode(OBFUSCATION_PEPPER)
	const bytes = new Uint8Array(mixed.length)

	for (let i = 0; i < mixed.length; i++) {
		bytes[i] = mixed[i] ^ pepper[i % pepper.length]
	}

	return decoder.decode(bytes)
}

const checksum = (value: string): string => {
	let hash = 0x811c9dc5

	for (const byte of encoder.encode(value)) {
		hash = Math.imul(hash ^ byte, 0x01000193) >>> 0
	}

	return hash.toString(16).padStart(8, '0')
}

const isStored = (raw: unknown): raw is StoredCredentials => {
	if (!raw || typeof raw !== 'object') return false

	const value = raw as Record<string, unknown>
	return (
		value.v === CREDENTIALS_VERSION &&
		typeof value.d === 'string' &&
		value.d !== '' &&
		typeof value.c === 'string' &&
		value.c !== '' &&
		typeof value.savedAt === 'number' &&
		Number.isFinite(value.savedAt)
	)
}

const isPayload = (raw: unknown): raw is CredentialsPayload => {
	if (!raw || typeof raw !== 'object') return false

	const value = raw as Record<string, unknown>
	return typeof value.u === 'string' && value.u !== '' && typeof value.p === 'string' && value.p !== ''
}

export const clearRememberedCredentials = () => {
	try {
		localStorage.removeItem(CREDENTIALS_KEY)
	} catch {
		return
	}
}

export const saveRememberedCredentials = (username: string, password: string) => {
	if (!username || !password) return

	const plain = JSON.stringify({ u: username, p: password })

	try {
		localStorage.setItem(
			CREDENTIALS_KEY,
			JSON.stringify({
				v: CREDENTIALS_VERSION,
				d: obfuscate(plain),
				c: checksum(plain),
				savedAt: Date.now(),
			})
		)
	} catch {
		return
	}
}

export const loadRememberedCredentials = (): IRememberedCredentials | null => {
	let raw: string | null
	try {
		raw = localStorage.getItem(CREDENTIALS_KEY)
	} catch {
		return null
	}
	if (!raw) return null

	let stored: unknown
	try {
		stored = JSON.parse(raw)
	} catch {
		clearRememberedCredentials()
		return null
	}

	if (!isStored(stored)) {
		clearRememberedCredentials()
		return null
	}

	const now = Date.now()
	if (stored.savedAt > now || now - stored.savedAt > CREDENTIALS_TTL_MS) {
		clearRememberedCredentials()
		return null
	}

	let plain: string
	try {
		plain = deobfuscate(stored.d)
	} catch {
		clearRememberedCredentials()
		return null
	}

	if (checksum(plain) !== stored.c) {
		clearRememberedCredentials()
		return null
	}

	let payload: unknown
	try {
		payload = JSON.parse(plain)
	} catch {
		clearRememberedCredentials()
		return null
	}

	if (!isPayload(payload)) {
		clearRememberedCredentials()
		return null
	}

	return { username: payload.u, password: payload.p }
}
