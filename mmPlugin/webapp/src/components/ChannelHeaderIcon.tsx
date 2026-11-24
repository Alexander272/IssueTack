import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

import {
	getContext,
	getContextFresh,
	getCachedContext,
	getCurrentTeamName,
	getCurrentUserId,
	getDmScope,
	requestGreeting,
	subscribeCurrentUser,
} from '../api'
import {
	rememberChannel,
	setModalHost,
	subscribeCreateSeed,
	subscribeOpen,
	subscribePluginOpen,
	takePendingTicket,
} from '../deepLink'
import type { CreateTicketSeed, PluginContextResult, PluginScope } from '../types'
import ModalController from './ModalController'
import { ClipboardCheckIcon } from './icons'

interface ChannelHeaderIconProps {
	channel: { id: string; type?: string; name?: string }
}

// botUserIdForDM возвращает собеседника личного диалога: имя DM-канала — это
// пара id через «__», из которой нужно выбрать чужую (не текущего пользователя).
// Для остальных каналов бота нет: реалм определяется по привязке канала.
function botUserIdForDM(channel: { type?: string; name?: string }, userId: string | null): string | undefined {
	if (!userId || channel.type !== 'D' || !channel.name) {
		return undefined
	}
	const ids = channel.name.split('__')
	if (ids.length !== 2) {
		return undefined
	}
	return ids[0] === userId ? ids[1] : ids[0]
}

// ChannelHeaderIcon — иконка в шапке личного диалога с ботом и заодно хост
// модалки плагина. Компонент смонтирован в шапке КАЖДОГО канала (MM рендерит
// ChannelHeaderIcon всегда), поэтому портал модалки рисуется из любого канала:
// иконку в шапке мы показываем только в DM с ботом, а модалку — по запросу
// иконки App Bar, даже когда сам компонент «пустой».
export default function ChannelHeaderIcon({ channel }: ChannelHeaderIconProps) {
	const channelId = channel ? channel.id : null
	const channelType = channel ? channel.type : undefined
	const channelName = channel ? channel.name : undefined
	const [userId, setUserId] = useState<string | null>(() => getCurrentUserId())
	// scope строится только для личных диалогов: иконка в шапке нужна лишь в DM
	// с ботом реалма, а публичные/приватные/GM-каналы к реалму не привязываются.
	const scope = useMemo<PluginScope | null>(() => {
		if (!channelId || !userId || channelType !== 'D') {
			return null
		}
		return { channelId, userId, botUserId: botUserIdForDM({ type: channelType, name: channelName }, userId) }
	}, [channelId, userId, channelType, channelName])
	const [context, setContext] = useState<PluginContextResult | null>(() => getCachedContext(scope))
	const [ready, setReady] = useState<boolean>(() => !!getCachedContext(scope))
	const [open, setOpen] = useState(false)
	// Заявка, которую надо показать сразу после открытия модалки: сюда её кладут
	// и перехватчик клика по ссылке, и TicketDeepLinkPage, когда уводит ссылку
	// из DM в канал (см. deepLink.ts).
	const [pendingOpenId, setPendingOpenId] = useState<string | null>(null)
	// scope/context, с которыми открыта модалка. Обычно это текущий DM с ботом,
	// но при клике иконки в App Bar из другого канала сюда кладётся DM-канал,
	// полученный с бэкенда: клиент id бота реалма не знает, а реалм
	// определяется по боту (см. PluginDM на бэкенде).
	const [modalScope, setModalScope] = useState<PluginScope | null>(null)
	const [modalContext, setModalContext] = useState<PluginContextResult | null>(null)
	// Предзаполнение из сообщения. Живёт до закрытия модалки: повторное
	// обычное открытие формы seed не подхватывает.
	const [createSeed, setCreateSeed] = useState<CreateTicketSeed | null>(null)
	const aliveRef = useRef(true)

	useEffect(() => {
		aliveRef.current = true
		const unsubscribe = subscribeCurrentUser(setUserId)
		return () => {
			aliveRef.current = false
			unsubscribe()
		}
	}, [])

	useEffect(() => {
		if (!scope) {
			setReady(false)
			return undefined
		}

		const fromCache = getCachedContext(scope)
		if (fromCache) {
			setContext(fromCache)
			setReady(true)
			return undefined
		}

		let cancelled = false
		setReady(false)
		getContext(scope)
			.then(data => {
				if (cancelled) {
					return
				}
				setContext(data)
				setReady(true)
			})
			.catch(() => {
				if (!cancelled) {
					setReady(false)
				}
			})

		return () => {
			cancelled = true
		}
	}, [scope])

	// Запоминаем канал, из которого открывают ссылки на заявки: маршрут плагина
	// живёт вне каналов, иначе вернуться в чат оттуда нечем. Только для scope,
	// то есть для DM с ботом, — в остальных каналах модалки всё равно нет.
	useEffect(() => {
		if (scope) {
			rememberChannel(getCurrentTeamName() || '', scope.channelId)
		}
	}, [scope])

	// Показываем ли мы сейчас модалку в текущем канале — по тем же условиям, что
	// и кнопка в шапке: флаг нужен перехватчику клика по ссылке, чтобы гасить
	// навигацию только когда заявку есть куда показать (см. deepLink.ts).
	const modalHost = Boolean(scope && ready && context?.bound)
	useEffect(() => {
		setModalHost(modalHost)
		return () => setModalHost(false)
	}, [modalHost])

	// Приветствие бота: пользователь открыл личный диалог. Ключ в sessionStorage
	// гасит повторные вызовы при переключении каналов в рамках сессии, а
	// окончательно повтор отсекает бэкенд (отметка «уже приветствовали» в БД).
	useEffect(() => {
		if (!modalHost || !scope) {
			return
		}
		const key = `it-plugin-greeted:${scope.userId}:${scope.botUserId || scope.channelId}`
		try {
			if (window.sessionStorage.getItem(key)) {
				return
			}
			window.sessionStorage.setItem(key, '1')
		} catch {
			// sessionStorage может быть недоступен — не мешаем вызову.
		}
		requestGreeting(scope).catch(() => undefined)
	}, [modalHost, scope])

	// refreshModalContext обновляет справочники уже открытой модалки: к моменту
	// клика они могли устареть, в том числе customers — по ним ищется автор
	// сообщения при предзаполнении заказчика.
	const refreshModalContext = useCallback((s: PluginScope) => {
		getContextFresh(s)
			.then(data => {
				if (aliveRef.current) {
					setModalContext(data)
				}
			})
			.catch(() => undefined)
	}, [])

	// openWith показывает модалку с заданным scope/контекстом. Вынесено до
	// раннего return, потому что на него подписываются иконка в App Bar,
	// перехватчик ссылок и иконка «Создать заявку» на посте.
	const openWith = useCallback((s: PluginScope, c: PluginContextResult, seed: CreateTicketSeed | null = null) => {
		setModalScope(s)
		setModalContext(c)
		setCreateSeed(seed)
		// Ссылка на заявку открывает вкладку «Мои заявки»; предзаполнение из
		// сообщения — всегда форму создания.
		if (seed) {
			setPendingOpenId(null)
		}
		setOpen(true)
	}, [])

	// openCurrent открывает модалку в текущем DM с ботом (иконка в шапке).
	const openCurrent = useCallback(() => {
		if (!scope || !context) {
			return
		}
		openWith(scope, context)
		refreshModalContext(scope)
	}, [scope, context, openWith, refreshModalContext])

	// openFromAnywhere — открытие из любого канала (App Bar или пост):
	// не уводим пользователя из текущего канала. Если мы уже в DM с ботом, scope
	// берём оттуда; иначе спрашиваем бэкенд, с каким ботом у пользователя личный
	// диалог, и работаем с этим DM-каналом. Пользователя может ещё не быть в
	// системе — его создаст /plugin/context по боту.
	const openFromAnywhere = useCallback(
		(seed: CreateTicketSeed | null = null) => {
			if (scope && context && context.bound) {
				openWith(scope, context, seed)
				refreshModalContext(scope)
				return
			}
			getDmScope()
				.then(dmScope => {
					if (!dmScope) {
						return null
					}
					return getContextFresh(dmScope).then(dmContext => ({ scope: dmScope, context: dmContext }))
				})
				.then(resolved => {
					if (resolved && aliveRef.current && resolved.context.bound) {
						openWith(resolved.scope, resolved.context, seed)
					}
				})
				.catch(() => undefined)
		},
		[scope, context, openWith, refreshModalContext],
	)

	// Клик по ссылке на заявку, когда модалка уже смонтирована: показываем её
	// прямо в текущем канале, вообще никуда не переходя.
	useEffect(
		() =>
			subscribeOpen(ticketId => {
				if (!scope || !context || !context.bound) {
					return
				}
				setPendingOpenId(ticketId)
				openWith(scope, context)
			}),
		[scope, context, openWith],
	)

	// Иконка в App Bar просит открыть модалку из любого канала.
	useEffect(() => subscribePluginOpen(() => openFromAnywhere()), [openFromAnywhere])

	// Иконка на посте просит открыть ту же модалку с предзаполнением.
	useEffect(() => subscribeCreateSeed(seed => openFromAnywhere(seed)), [openFromAnywhere])

	// Ссылка на заявку положила её id в deepLink (запасной путь через маршрут
	// плагина). Забираем её здесь — только когда модалка действительно сможет
	// открыться (bound), иначе запись сгорела бы впустую вместе с единственным
	// шансом её показать.
	useEffect(() => {
		if (!ready || !context?.bound || open || pendingOpenId || !scope) {
			return
		}
		const pending = takePendingTicket()
		if (pending) {
			setPendingOpenId(pending)
			openWith(scope, context)
		}
	}, [ready, context?.bound, open, pendingOpenId, scope, context, openWith])

	const showButton = Boolean(scope && ready && context && context.bound)

	return (
		<>
			{showButton && scope ? (
				<button
					type='button'
					className='it-ticket-header-icon channel-header__icon channel-header__icon--wide channel-header__icon--left'
					aria-label='Заявки в службу IT'
					title='Заявки в службу IT'
					onClick={openCurrent}
				>
					<ClipboardCheckIcon size={16} />
					<span className='it-ticket-header-icon__label'>Заявки</span>
				</button>
			) : null}
			{open && modalScope && modalContext
				? createPortal(
						<ModalController
							scope={modalScope}
							context={modalContext}
							initialPendingOpenId={pendingOpenId}
							initialCreateSeed={createSeed}
							onClose={() => {
								setOpen(false)
								setPendingOpenId(null)
								setCreateSeed(null)
							}}
						/>,
						document.body,
					)
				: null}
		</>
	)
}
