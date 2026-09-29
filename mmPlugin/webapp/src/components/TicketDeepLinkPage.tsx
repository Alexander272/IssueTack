import React, { useCallback, useEffect, useState } from 'react'

import { ApiError, getCurrentUserId, getTicketLinkContext, subscribeCurrentUser } from '../api'
import TicketDetail from './TicketDetail'

// TicketDeepLinkPage — страница заявки, открытая по ссылке
// /plug/issuetrack/ticket/<uuid> из DM-уведомления или подтверждения создания.
//
// Особенности, из-за которых страница отдельная от модалки:
//  1. Компонент, зарегистрированный через registerCustomRoute, не получает
//     props — ни channel, ни match. Идентификатор заявки берётся из location.
//  2. Маршрут не привязан к каналу, поэтому текущего канала в клиенте нет:
//     реалм и нужный для запросов channelID определяет сервер по самой заявке.
//  3. Обёртки LoggedIn у маршрута нет, поэтому отсутствие пользователя
//     обрабатывается явно.

const TICKET_ID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

function readTicketIdFromPath(): string | null {
    // Ищем хвост маршрута, а не сравниваем pathname целиком: установка может
    // быть под субпатом, и путь будет вида /mattermost/plug/issuetrack/ticket/<uuid>.
    const marker = '/plug/issuetrack/ticket/'
    const path = window.location.pathname || ''
    const at = path.indexOf(marker)
    if (at === -1) {
        return null
    }
    const rest = path.slice(at + marker.length)
    const raw = (rest.split('/')[0] || '').split('?')[0]
    const decoded = (() => {
        try {
            return decodeURIComponent(raw)
        } catch (e) {
            return raw
        }
    })()
    return TICKET_ID_RE.test(decoded) ? decoded : null
}

export default function TicketDeepLinkPage() {
    const [ticketId] = useState<string | null>(() => readTicketIdFromPath())
    const [userId, setUserId] = useState<string | null>(() => getCurrentUserId())
    const [channelId, setChannelId] = useState<string | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [loading, setLoading] = useState(true)

    // Пользователь появляется в сторе не сразу при полной перезагрузке страницы,
    // поэтому подписываемся на его появление вместо однократного чтения.
    useEffect(() => subscribeCurrentUser(setUserId), [])

    useEffect(() => {
        if (!ticketId) {
            setError('Ссылка на заявку не распознана: проверьте, что она скопирована полностью.')
            setLoading(false)
            return
        }
        if (!userId) {
            return
        }

        let cancelled = false
        setLoading(true)
        setError(null)
        getTicketLinkContext(userId, ticketId)
            .then(ctx => {
                if (cancelled) {
                    return
                }
                setChannelId(ctx.channelId)
                setLoading(false)
            })
            .catch((err: unknown) => {
                if (cancelled) {
                    return
                }
                setChannelId(null)
                setError(err instanceof ApiError ? err.message : 'Не удалось загрузить заявку')
                setLoading(false)
            })

        return () => {
            cancelled = true
        }
    }, [ticketId, userId])

    const handleBack = useCallback(() => {
        // Возврат в предыдущее место: обычно это канал, из которого пришла
        // ссылка в DM. Если истории нет (страница открыта напрямую), уходим
        // на корень Mattermost.
        if (window.history.length > 1) {
            window.history.back()
        } else {
            window.location.assign('/')
        }
    }, [])

    // TicketDetail сам перезапрашивает заявку после действий, а список заявок
    // на этой странице отсутствует — обновлять нечего, поэтому no-op.
    const handleChanged = useCallback(() => undefined, [])

    if (!userId) {
        return (
            <div className='it-ticket-page'>
                <div className='it-ticket-page__notice'>
                    Чтобы открыть заявку, войдите в Mattermost.
                </div>
                <div className='it-ticket-page__notice-actions'>
                    <button type='button' className='it-btn it-btn--sm' onClick={handleBack}>
                        ← Назад
                    </button>
                </div>
            </div>
        )
    }

    return (
        <div className='it-ticket-page'>
            {error ? (
                <>
                    <div className='it-ticket-page__notice it-ticket-page__notice--error'>{error}</div>
                    <div className='it-ticket-page__notice-actions'>
                        <button type='button' className='it-btn it-btn--sm' onClick={handleBack}>
                            ← Назад
                        </button>
                    </div>
                </>
            ) : null}

            {loading && !channelId && !error ? <div className='it-ticket-detail__loading'>Загрузка…</div> : null}

            {channelId && ticketId ? (
                <TicketDetail
                    channelId={channelId}
                    userId={userId}
                    ticketId={ticketId}
                    onBack={handleBack}
                    onChanged={handleChanged}
                    backLabel='← Назад'
                />
            ) : null}
        </div>
    )
}
