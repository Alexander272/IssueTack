import { wsService } from '@/app/services/socket'
import { useEffect, useState } from 'react'

export const ConnectionStatus = () => {
	// Инициализируем статус на основе реального состояния сервиса
	const [status, setStatus] = useState<'connected' | 'reconnecting'>(
		wsService.isConnected() ? 'connected' : 'reconnecting',
	)

	useEffect(() => {
		// 1. Слушаем успешное подключение
		const unSubConnect = wsService.subscribe('SYSTEM_CONNECTED', () => {
			setStatus('connected')
		})

		// 2. Слушаем начало реконнекта
		const unSubReconnect = wsService.subscribe('SYSTEM_RECONNECTING', () => {
			setStatus('reconnecting')
		})

		// 3. Слушаем обрыв связи
		const unSubDisconnect = wsService.subscribe('SYSTEM_DISCONNECTED', () => {
			setStatus('reconnecting')
		})

		// Дополнительная проверка "на всякий случай" каждые 5 сек
		const checkInterval = setInterval(() => {
			const connected = wsService.isConnected()
			if (connected && status !== 'connected') setStatus('connected')
			if (!connected && status !== 'reconnecting') setStatus('reconnecting')
		}, 5000)

		return () => {
			unSubConnect()
			unSubReconnect()
			unSubDisconnect()
			clearInterval(checkInterval)
		}
	}, [status])

	// Если всё хорошо — ничего не показываем
	if (status === 'connected') return null

	return (
		<div
			style={{
				position: 'fixed',
				bottom: '20px',
				right: '20px',
				padding: '10px 20px',
				borderRadius: '8px',
				backgroundColor: status === 'reconnecting' ? '#f59e0b' : '#ef4444',
				color: 'white',
				boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
				zIndex: 9999,
				display: 'flex',
				alignItems: 'center',
				gap: '10px',
			}}
		>
			<div className='spinner' />
			{status === 'reconnecting' ? 'Восстановление связи...' : 'Ошибка соединения'}
		</div>
	)
}
