import React, {useEffect, useMemo, useRef, useState} from 'react'
import {createPortal} from 'react-dom'

import {
    getContext,
    getContextFresh,
    getCachedContext,
    getCurrentTeamName,
    getCurrentUserId,
    subscribeCurrentUser,
} from '../api'
import {rememberChannel, setModalHost, subscribeOpen, takePendingTicket} from '../deepLink'
import type {PluginContextResult, PluginScope} from '../types'
import ModalController from './ModalController'
import {ClipboardCheckIcon} from './icons'

interface ChannelHeaderIconProps {
    channel: {id: string; type?: string; name?: string}
}

// botUserIdForDM возвращает собеседника личного диалога: имя DM-канала — это
// пара id через «__», из которой нужно выбрать чужую (не текущего пользователя).
// Для остальных каналов бота нет: реалм определяется по привязке канала.
function botUserIdForDM(channel: {type?: string; name?: string}, userId: string | null): string | undefined {
    if (!userId || channel.type !== 'D' || !channel.name) {
        return undefined;
    }
    const ids = channel.name.split('__');
    if (ids.length !== 2) {
        return undefined;
    }
    return ids[0] === userId ? ids[1] : ids[0];
}

export default function ChannelHeaderIcon({channel}: ChannelHeaderIconProps) {
    const channelId = channel ? channel.id : null
    const channelType = channel ? channel.type : undefined
    const channelName = channel ? channel.name : undefined
    const [userId, setUserId] = useState<string | null>(() => getCurrentUserId())
    // scope строится только для личных диалогов: иконка нужна лишь в DM с ботом
    // реалма, а публичные/приватные/GM-каналы к реалму не привязываются.
    const scope = useMemo<PluginScope | null>(() => {
        if (!channelId || !userId || channelType !== 'D') {
            return null
        }
        return {channelId, userId, botUserId: botUserIdForDM({type: channelType, name: channelName}, userId)}
    }, [channelId, userId, channelType, channelName])
    const [context, setContext] = useState<PluginContextResult | null>(() => getCachedContext(scope))
    const [ready, setReady] = useState<boolean>(() => !!getCachedContext(scope))
    const [open, setOpen] = useState(false)
    // Заявка, которую надо показать сразу после открытия модалки: сюда её кладут
    // и перехватчик клика по ссылке, и TicketDeepLinkPage, когда уводит ссылку
    // из DM в канал (см. deepLink.ts).
    const [pendingOpenId, setPendingOpenId] = useState<string | null>(null)
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
            rememberChannel(getCurrentTeamName() || '', scope.channelId);
        }
    }, [scope?.channelId]);

    // Показываем ли мы сейчас модалку — по тем же условиям, что и рендер: флаг
    // нужен перехватчику клика по ссылке, чтобы гасить навигацию только когда
    // заявку есть куда показать (см. deepLink.ts).
    const modalHost = Boolean(scope && ready && context?.bound);
    useEffect(() => {
        setModalHost(modalHost);
        return () => setModalHost(false);
    }, [modalHost]);

    // Клик по ссылке на заявку, когда модалка уже смонтирована: показываем её
    // прямо в текущем канале, вообще никуда не переходя.
    useEffect(() => subscribeOpen(ticketId => {
        setPendingOpenId(ticketId);
        setOpen(true);
    }), []);

    // Ссылка на заявку положила её id в deepLink и ушла в канал. Забираем его
    // здесь — только когда модалка действительно сможет открыться (bound), иначе
    // запись сгорела бы впустую вместе с единственным шансом её показать.
    useEffect(() => {
        if (!ready || !context?.bound || open || pendingOpenId) {
            return
        }
        const pending = takePendingTicket();
        if (pending) {
            setPendingOpenId(pending)
            setOpen(true)
        }
    }, [ready, context?.bound, open, pendingOpenId])

    if (!scope || !ready || !context || !context.bound) {
        return null
    }

    const openModal = () => {
        setOpen(true)
        if (!scope) {
            return
        }
        getContextFresh(scope)
            .then(data => {
                if (aliveRef.current) {
                    setContext(data)
                }
            })
            .catch(() => undefined)
    }

    return (
        <>
            <button
                type='button'
                className='it-ticket-header-icon channel-header__icon channel-header__icon--wide channel-header__icon--left'
                aria-label='Заявки в IT отдел'
                title='Заявки в IT отдел'
                onClick={openModal}
            >
                <ClipboardCheckIcon size={16} />
                <span className='it-ticket-header-icon__label'>Заявки</span>
            </button>
            {open && scope
                ? createPortal(
                        <ModalController
                            scope={scope}
                            context={context}
                            initialPendingOpenId={pendingOpenId}
                            onClose={() => {
                                setOpen(false);
                                setPendingOpenId(null);
                            }}
                        />,
                        document.body,
                    )
                : null}
        </>
    )
}