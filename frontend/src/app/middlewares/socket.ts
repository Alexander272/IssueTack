import type { Middleware, PayloadAction } from '@reduxjs/toolkit'
import type { ISocketEnvelope } from '../types/socket'

// Типы событий для внутреннего использования
// Экшены для управления сокетом
export const socketSend = (payload: ISocketEnvelope): PayloadAction<ISocketEnvelope> => ({
	type: 'socket/send',
	payload,
})

export const socketMessageReceived = (payload: ISocketEnvelope): PayloadAction<ISocketEnvelope> => ({
	type: 'socket/message',
	payload,
})

export const socketMiddleware: Middleware = store => {
	let socket: WebSocket | null = null

	return next => action => {
		// Сужаем тип action через проверку
		const socketAction = action as PayloadAction<ISocketEnvelope>

		if (!socket && typeof window !== 'undefined') {
			const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
			socket = new WebSocket(`${protocol}//${window.location.host}/api/ws`)

			socket.onmessage = (event: MessageEvent) => {
				try {
					const message: ISocketEnvelope = JSON.parse(event.data)
					store.dispatch(socketMessageReceived(message))
				} catch (e) {
					console.error('Socket parse error', e)
				}
			}
		}

		if (socketAction.type === 'socket/send' && socket?.readyState === WebSocket.OPEN) {
			socket.send(JSON.stringify(socketAction.payload))
			return
		}

		return next(action)
	}
}
