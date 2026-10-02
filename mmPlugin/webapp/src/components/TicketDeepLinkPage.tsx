import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { ApiError, getCurrentTeamName, getCurrentUserId, getTicketLinkContext, subscribeCurrentUser } from '../api'
import { channelPath, navigateClientSide, reloadTo, restoreChannel, setPendingTicket, takeChannel } from '../deepLink'
import type { PluginScope } from '../types'
import TicketDetail from './TicketDetail'

// TicketDeepLinkPage — обработчик ссылки /plug/issuetrack/ticket/<uuid> из
// DM-уведомления или подтверждения создания.
//
// Основной путь — не показывать страницу, а уйти в канал и открыть заявку в
// модалке поверх чата: на маршруте плагина Mattermost монтирует этот компонент
// как единственный вид, поэтому ни меню каналов, ни хоста модалки там нет.
// Канал берётся из sessionStorage (см. deepLink.ts), туда же кладётся id заявки;
// иконка в шапке канала поднимает её и открывает модалку.
//
// Если канал неизвестен (например, ссылку открыли в новой вкладке, где плагин
// ещё ни разу не монтировался) — показывается страница заявки: отдельная от
// модалки, потому что канала у неё нет и взять его неоткуда.
//
// Особенности, из-за которых страница отдельная от модалки:
//  1. Компонент, зарегистрированный через registerCustomRoute, не получает
//     props — ни channel, ни match. Идентификатор заявки берётся из location.
//  2. Маршрут не привязан к каналу, поэтому текущего канала в клиенте нет:
//     реалм и нужный для запросов channelID определяет сервер по самой заявке.
//  3. Обёртки LoggedIn у маршрута нет, поэтому отсутствие пользователя
//     обрабатывается явно.

// Сколько ждём реакции react-router на переход, прежде чем перезагрузить
// страницу принудительно: клиентский переход мгновенный, а location.assign
// теряет весь открытый Mattermost.
const CLIENT_NAVIGATION_TIMEOUT_MS = 300;

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
    // Страница открыта вне канала, поэтому бота диалога здесь нет: scope
    // строится только из канала реалма и текущего пользователя. Мемоизируется,
    // иначе новый объект на каждом рендере перезапускал бы загрузку заявки.
    const scope = useMemo<PluginScope | null>(() => (channelId && userId ? { channelId, userId } : null), [channelId, userId])
    const [error, setError] = useState<string | null>(null)
    const [loading, setLoading] = useState(true)
    // Пока идёт переход в канал, ничего не рисуем и грузить заявку не нужно.
    const [redirecting, setRedirecting] = useState(false)
    const aliveRef = useRef(true)
    const redirectTimerRef = useRef<number | null>(null)

    // Таймер намеренно не чистится в cleanup эффекта перехода: тот перезапускается
    // на setRedirecting(true) и погасил бы таймер раньше, чем он сработает.
    // Гасим его только при размонтировании.
    useEffect(() => {
        aliveRef.current = true
        return () => {
            aliveRef.current = false
            if (redirectTimerRef.current !== null) {
                window.clearTimeout(redirectTimerRef.current)
                redirectTimerRef.current = null
            }
        }
    }, [])

    // Уход в канал. Канал запомнен иконкой в шапке (deepLink.rememberChannel)
    // вместе с именем команды; без обоих показываем страницу как раньше.
    useEffect(() => {
        if (!ticketId || redirecting) {
            return
        }
        const remembered = takeChannel()
        // Имя команды могло не попасть в запомненное (стор ещё не гидратирован,
        // когда монтировался DM) — тогда берём текущую.
        const teamName = remembered?.teamName || getCurrentTeamName() || ''
        const target = remembered ? channelPath(teamName, remembered.channelId) : null
        if (!target) {
            // Канал был, но адрес не собрался (команда не гидратирована) — вернём
            // запись на место, иначе вторая ссылка открылась бы страницей.
            if (remembered) {
                restoreChannel(remembered)
            }
            return
        }
        setRedirecting(true)
        setPendingTicket(ticketId)
        navigateClientSide(target)
        redirectTimerRef.current = window.setTimeout(() => {
            // Если нас всё ещё монтируют — react-router переход не сработал
            // и канал не отрисован. Тогда грузим с перезагрузкой: медленнее,
            // зато точно открывается.
            if (aliveRef.current) {
                reloadTo(target)
            }
        }, CLIENT_NAVIGATION_TIMEOUT_MS)
    }, [ticketId, redirecting])

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
        // Идём в канал — заявку здесь грузить незачем, её покажет модалка.
        if (redirecting) {
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
    }, [ticketId, userId, redirecting])

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

    // Пока идёт переход в канал, экран пустой: рисовать тут нечего, а любой
    // контент мелькнёт перед тем, как Mattermost перерисует вид канала.
    if (redirecting) {
        return null
    }

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

            {loading && !scope && !error ? <div className='it-ticket-detail__loading'>Загрузка…</div> : null}

            {scope && ticketId ? (
                <TicketDetail
                    scope={scope}
                    ticketId={ticketId}
                    onBack={handleBack}
                    onChanged={handleChanged}
                    backLabel='← Назад'
                />
            ) : null}
        </div>
    )
}
