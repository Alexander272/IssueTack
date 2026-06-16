import type { IOrder, IOrderSocketMessage } from '@/features/orders/types/order'
import type { IOrderMatchResult, ISearchError, ISearchItem } from '@/features/search/types/search'

// Базовый интерфейс сообщения
export interface ISocketEnvelope<T = unknown> {
	action: string
	data: T
}

// Типы входящих действий (из твоего Go Hub)
export type TServerAction = 'INSERT' | 'UPDATE' | 'DELETE' | 'INSERT_MANY'

export type OrderAction = 'INSERT' | 'UPDATE' | 'DELETE' | 'INSERT_MANY'
export type SearchAction = 'SEARCH_STREAM' | 'SEARCH_RESULT' | 'SEARCH_RESULT_PART'

export interface BaseWSMessage {
	type: string
}

export interface OrderWSMessage extends BaseWSMessage {
	action: OrderAction
	data: IOrderSocketMessage
}

export interface SearchWSMessage extends BaseWSMessage {
	action: SearchAction
	data: IOrderMatchResult[]
}

export type WSMessage = OrderWSMessage | SearchWSMessage

export type WSEvent =
	| { action: 'SYSTEM_CONNECTED'; data: null }
	| { action: 'SYSTEM_DISCONNECTED'; data: null }
	| { action: 'SYSTEM_RECONNECTING'; data: null }
	| { action: 'ORDER_INSERTED'; data: IOrder }
	| { action: 'ORDER_UPDATED'; data: IOrder }
	| { action: 'ORDER_DELETED'; data: { id: string; year: number } }
	| { action: 'ORDERS_BULK_INSERTED'; data: { years: number[] } }
	| { action: 'SEARCH_STREAM'; data: ISearchItem[] }
	| { action: 'SEARCH_RESULT'; data: IOrderMatchResult[] }
	| { action: 'SEARCH_RESULT_PART'; data: { items: IOrderMatchResult[]; isLast: boolean; total: number } }
	| { action: 'SEARCH_ERROR'; data: ISearchError }
	| { action: 'CANCEL_SEARCH'; data: null }
	| { action: 'SUBSCRIBE'; data: null }
	| { action: 'UNSUBSCRIBE'; data: null }

// Превращаем Union в Map для сервиса
export type WSEventMap = {
	[E in WSEvent as E['action']]: E['data']
}

export type Listener<T> = (data: T) => void
