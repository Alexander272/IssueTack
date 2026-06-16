import type { Listener, WSEvent, WSEventMap } from '../types/socket'

// class WebSocketService {
// 	private ws: WebSocket | null = null

// 	private listeners: Partial<Record<keyof WSEventMap, Set<Listener<unknown>>>> = {}

// 	private reconnectTimeout: ReturnType<typeof setTimeout> | null = null
// 	private isClosedManually = false

// 	connect() {
// 		if (this.ws) return

// 		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
// 		this.ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)

// 		this.ws.onmessage = (event: MessageEvent<string>) => {
// 			const parsed: unknown = JSON.parse(event.data)

// 			if (!this.isWSMessage(parsed)) return

// 			const handlers = this.listeners[parsed.type]
// 			if (!handlers) return

// 			handlers.forEach(cb => {
// 				cb(parsed)
// 			})
// 		}

// 		this.ws.onclose = () => {
// 			this.ws = null

// 			if (!this.isClosedManually) {
// 				this.reconnectTimeout = setTimeout(() => this.connect(), 5000)
// 			}
// 		}

// 		this.ws.onerror = () => {
// 			this.ws?.close()
// 		}
// 	}

// 	subscribe<K extends keyof WSEventMap>(type: K, cb: Listener<WSEventMap[K]>): () => void {
// 		if (!this.listeners[type]) {
// 			this.listeners[type] = new Set()
// 		}

// 		const set = this.listeners[type] as Set<Listener<WSEventMap[K]>>

// 		set.add(cb)

// 		return () => {
// 			set.delete(cb)
// 		}
// 	}

// 	close() {
// 		this.isClosedManually = true
// 		if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout)
// 		this.ws?.close(1000, 'Manual close')
// 	}

// 	private isWSMessage(data: unknown) {
// 		return data && typeof data === 'object' && 'type' in data
// 	}
// 	// private isWSMessage(data: unknown): data is WSEventMap[keyof WSEventMap] {
// 	// 	if (typeof data !== 'object' || data === null) return false
// 	// 	if (!('type' in data)) return false

// 	// 	const t = (data as { type?: unknown }).type
// 	// 	return t === 'ORDER_EVENT' || t === 'SEARCH_RESULT'
// 	// }
// }

class WebSocketService {
	private ws: WebSocket | null = null
	private listeners: Partial<{ [K in keyof WSEventMap]: Set<Listener<WSEventMap[K]>> }> = {}

	private reconnectAttempts = 0
	private reconnectTimeout: ReturnType<typeof setTimeout> | null = null
	private isClosedManually = false

	private emit<K extends keyof WSEventMap>(type: K, payload: WSEventMap[K]) {
		const handlers = this.listeners[type]
		if (handlers) {
			// Используем приведение к unknown для обхода строгой типизации, как мы делали раньше
			const genericHandlers = handlers as unknown as Set<Listener<unknown>>
			genericHandlers.forEach(cb => cb(payload))
		}
	}

	public isConnected(): boolean {
		return this.ws !== null && this.ws.readyState === WebSocket.OPEN
	}

	connect() {
		if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
			return
		}

		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
		this.ws = new WebSocket(`${protocol}//${window.location.host}/api/ws`)
		console.log('📡 WS Connecting...')

		this.ws.onopen = () => {
			console.log('✅ WS Connected')
			this.reconnectAttempts = 0
			this.emit('SYSTEM_CONNECTED', null)
		}

		this.ws.onmessage = (event: MessageEvent<string>) => {
			// console.log('📥 WS Raw Data:', event.data)
			try {
				const data = JSON.parse(event.data) as WSEvent
				if (!data.action) return

				const handlers = this.listeners[data.action]
				if (handlers) {
					const genericHandlers = handlers as unknown as Set<Listener<unknown>>

					genericHandlers.forEach(cb => {
						cb(data.data)
					})
				}
			} catch (e) {
				console.error('WS Error', e)
			}
		}

		this.ws.onclose = () => {
			this.ws = null
			if (!this.isClosedManually) this.planReconnect()
		}

		this.ws.onerror = () => this.ws?.close()
	}

	send<K extends keyof WSEventMap>(action: K, payload: unknown) {
		console.log('📡 WS Sending:', action, payload)
		if (this.ws?.readyState === WebSocket.OPEN) {
			this.ws.send(JSON.stringify({ action, payload }))
		}
	}

	subscribe<K extends keyof WSEventMap>(type: K, cb: Listener<WSEventMap[K]>): () => void {
		if (!this.listeners[type]) {
			// Сначала приводим к unknown, потом к типу, который ожидает Record
			this.listeners[type] = new Set<Listener<WSEventMap[K]>>() as unknown as Partial<{
				[P in keyof WSEventMap]: Set<Listener<WSEventMap[P]>>
			}>[K]
		}

		// Извлекаем через unknown, чтобы TS "забыл" о своих подозрениях
		const handlers = this.listeners[type] as unknown as Set<Listener<WSEventMap[K]>>

		handlers.add(cb)

		return () => {
			handlers.delete(cb)
			if (handlers.size === 0) {
				delete this.listeners[type]
			}
		}
	}

	private planReconnect() {
		if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout)
		const delay = Math.min(30000, 1000 * Math.pow(2, this.reconnectAttempts))

		this.emit('SYSTEM_RECONNECTING', null)
		console.log('📡 WS Reconnecting in', delay, 'seconds')

		this.reconnectTimeout = setTimeout(() => {
			this.reconnectAttempts++
			this.connect()
		}, delay)
	}

	close() {
		this.isClosedManually = true
		if (this.reconnectTimeout) clearTimeout(this.reconnectTimeout)

		if (!this.ws) return
		console.log('📡 WS Closing...')

		if (this.ws.readyState === WebSocket.CONNECTING) {
			// Если сокет еще подключается, вешаем одноразовый обработчик на закрытие сразу после открытия
			this.ws.onopen = () => {
				this.ws?.close(1000, 'Closed after pending connect')
				this.ws = null
			}
		} else if (this.ws.readyState === WebSocket.OPEN) {
			this.ws.close(1000, 'Manual close')
			this.emit('SYSTEM_DISCONNECTED', null)
			this.ws = null
		}
	}
}

export const wsService = new WebSocketService()
