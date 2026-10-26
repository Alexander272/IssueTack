import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { ApiError, getMyTickets } from '../api'
import type { PluginScope, PluginTicketShort } from '../types'
import { useDebounce } from '../useDebounce'
import TicketList from './TicketList'
import TicketDetail from './TicketDetail'

interface MyTicketsTabProps {
    scope: PluginScope
    // pendingOpenId — id заявки, которую нужно открыть сразу (экран успеха формы
    // создания). После открытия onPendingOpenHandled сбрасывает его, чтобы
    // повторный рендер не переоткрывал карточку.
    pendingOpenId?: string | null
    onPendingOpenHandled?: () => void
}

type TabMode = 'active' | 'archive'

interface ArchiveFilters {
    search: string
    closedAtFrom: string
    closedAtTo: string
}

const PAGE_SIZE = 10
const SEARCH_DEBOUNCE_MS = 300

const formatDay = (d: Date) => {
    const month = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${d.getFullYear()}-${month}-${day}`
}

// defaultArchiveFilters — окно по умолчанию: последние 30 дней. Без него
// вкладка открывалась бы на всей закрытой истории пользователя.
const defaultArchiveFilters = (): ArchiveFilters => {
    const to = new Date()
    const from = new Date(to)
    from.setDate(from.getDate() - 30)
    return {
        search: '',
        closedAtFrom: formatDay(from),
        closedAtTo: formatDay(to),
    }
}

export default function MyTicketsTab({ scope, pendingOpenId, onPendingOpenHandled }: MyTicketsTabProps) {
    const [mode, setMode] = useState<TabMode>('active')
    const [selectedId, setSelectedId] = useState<string | null>(null)

    const [activeTickets, setActiveTickets] = useState<PluginTicketShort[] | null>(null)
    const [activeError, setActiveError] = useState<string | null>(null)
    const activeReqRef = useRef(0)

    const [archiveItems, setArchiveItems] = useState<PluginTicketShort[]>([])
    const [archiveTotal, setArchiveTotal] = useState(0)
    const [archiveError, setArchiveError] = useState<string | null>(null)
    const [archiveLoading, setArchiveLoading] = useState(false)
    const [archiveLoadingMore, setArchiveLoadingMore] = useState(false)
    const [archiveFilters, setArchiveFilters] = useState<ArchiveFilters>(defaultArchiveFilters)
    const archiveReqRef = useRef(0)
    // archiveLoadedKeyRef — запрос, для которого архив уже загружен. Гасит
    // повторную загрузку при возврате на вкладку, но не мешает смене фильтров.
    const archiveLoadedKeyRef = useRef<string | null>(null)

    // Дебаунсится только сетевой запрос: поле ввода обновляется сразу.
    const debouncedSearch = useDebounce(archiveFilters.search, SEARCH_DEBOUNCE_MS)

    const archiveQuery = useMemo(
        () => ({
            search: debouncedSearch.trim(),
            closedAtFrom: archiveFilters.closedAtFrom,
            closedAtTo: archiveFilters.closedAtTo,
        }),
        [debouncedSearch, archiveFilters.closedAtFrom, archiveFilters.closedAtTo],
    )

    const archiveQueryKey = `${archiveQuery.search}|${archiveQuery.closedAtFrom}|${archiveQuery.closedAtTo}`

    const loadActive = useCallback(() => {
        const reqId = ++activeReqRef.current
        setActiveTickets(null)
        setActiveError(null)
        getMyTickets(scope)
            .then(result => {
                if (reqId === activeReqRef.current) {
                    setActiveTickets(result?.items || [])
                }
            })
            .catch((err: unknown) => {
                if (reqId === activeReqRef.current) {
                    setActiveError(err instanceof ApiError ? err.message : 'Не удалось загрузить заявки')
                }
            })
    }, [scope])

    useEffect(() => {
        loadActive()
    }, [loadActive])

    // loadArchive ходит на сервер: фильтрацию по заголовку и номеру делает
    // TicketFilter.Search (тот же SQL, что и в веб-списке заявок), а не клиент.
    const loadArchive = useCallback(
        (reset: boolean, query: ArchiveFilters, offset: number) => {
            const reqId = ++archiveReqRef.current
            if (reset) {
                setArchiveLoading(true)
                setArchiveItems([])
                setArchiveTotal(0)
                setArchiveError(null)
            } else {
                setArchiveLoadingMore(true)
            }

            getMyTickets(scope, {
                archived: true,
                limit: PAGE_SIZE,
                offset,
                search: query.search || undefined,
                closedAtFrom: query.closedAtFrom || undefined,
                closedAtTo: query.closedAtTo || undefined,
            })
                .then(result => {
                    if (reqId !== archiveReqRef.current) {
                        return
                    }
                    const items = result?.items || []
                    setArchiveTotal(result?.total || 0)
                    setArchiveItems(prev => (reset ? items : prev.concat(items)))
                    setArchiveError(null)
                })
                .catch((err: unknown) => {
                    if (reqId !== archiveReqRef.current) {
                        return
                    }
                    // Снимаем отметку загрузки: иначе после ошибки повторный
                    // заход на вкладку не попробовал бы запрос снова.
                    archiveLoadedKeyRef.current = null
                    setArchiveError(err instanceof ApiError ? err.message : 'Не удалось загрузить архив')
                })
                .finally(() => {
                    if (reqId !== archiveReqRef.current) {
                        return
                    }
                    setArchiveLoading(false)
                    setArchiveLoadingMore(false)
                })
        },
        [scope],
    )

    useEffect(() => {
        if (mode !== 'archive') {
            return
        }
        if (archiveLoadedKeyRef.current === archiveQueryKey) {
            return
        }
        archiveLoadedKeyRef.current = archiveQueryKey
        loadArchive(true, archiveQuery, 0)
    }, [archiveQuery, archiveQueryKey, loadArchive, mode])

    useEffect(() => {
        if (!pendingOpenId) {
            return
        }
        setSelectedId(pendingOpenId)
        onPendingOpenHandled?.()
    }, [pendingOpenId, onPendingOpenHandled])

    const onRefresh = useCallback(() => {
        if (mode === 'active') {
            loadActive()
            return
        }
        archiveLoadedKeyRef.current = archiveQueryKey
        loadArchive(true, archiveQuery, 0)
    }, [archiveQuery, archiveQueryKey, loadActive, loadArchive, mode])

    const onFiltersChange = useCallback((patch: Partial<ArchiveFilters>) => {
        setArchiveFilters(prev => ({ ...prev, ...patch }))
    }, [])

    const onLoadMore = useCallback(() => {
        if (archiveLoadingMore || archiveLoading) {
            return
        }
        if (archiveItems.length >= archiveTotal) {
            return
        }
        loadArchive(false, archiveQuery, archiveItems.length)
    }, [archiveItems.length, archiveLoading, archiveLoadingMore, archiveQuery, archiveTotal, loadArchive])

    if (selectedId) {
        return (
            <TicketDetail
                key={selectedId}
                scope={scope}
                ticketId={selectedId}
                onBack={() => setSelectedId(null)}
                onChanged={onRefresh}
            />
        )
    }

    return (
        <div className='it-my-tickets'>
            <div className='it-my-tickets__tabs'>
                <button
                    type='button'
                    className={`it-my-tickets__tab${mode === 'active' ? ' it-my-tickets__tab--active' : ''}`}
                    onClick={() => setMode('active')}
                >
                    Активные
                </button>
                <button
                    type='button'
                    className={`it-my-tickets__tab${mode === 'archive' ? ' it-my-tickets__tab--active' : ''}`}
                    onClick={() => setMode('archive')}
                >
                    Архив
                </button>
            </div>

            {mode === 'active' ? (
                <TicketList
                    tickets={activeTickets}
                    error={activeError}
                    onOpen={id => setSelectedId(id)}
                    onRefresh={loadActive}
                    emptyText='Заявок пока нет.'
                />
            ) : (
                <TicketList
                    tickets={archiveLoading && archiveItems.length === 0 ? null : archiveItems}
                    error={archiveError}
                    onOpen={id => setSelectedId(id)}
                    onRefresh={onRefresh}
                    emptyText='В архиве ничего не найдено.'
                    mode='archive'
                    filters={archiveFilters}
                    onFiltersChange={onFiltersChange}
                    total={archiveTotal}
                    hasMore={!archiveLoading && archiveItems.length < archiveTotal}
                    loadingMore={archiveLoadingMore}
                    onLoadMore={onLoadMore}
                />
            )}
        </div>
    )
}
