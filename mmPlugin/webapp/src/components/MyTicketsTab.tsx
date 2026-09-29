import React, { useCallback, useEffect, useRef, useState } from 'react'

import { ApiError, getMyTickets } from '../api'
import type { PluginScope, PluginTicketShort } from '../types'
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

export default function MyTicketsTab({ scope, pendingOpenId, onPendingOpenHandled }: MyTicketsTabProps) {
    const [tickets, setTickets] = useState<PluginTicketShort[] | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [selectedId, setSelectedId] = useState<string | null>(null)
    const reqRef = useRef(0)

    const load = useCallback(() => {
        const reqId = ++reqRef.current
        setTickets(null)
        setError(null)
        getMyTickets(scope)
            .then(list => {
                if (reqId === reqRef.current) {
                    setTickets(list || [])
                }
            })
            .catch((err: unknown) => {
                if (reqId === reqRef.current) {
                    setError(err instanceof ApiError ? err.message : 'Не удалось загрузить заявки')
                }
            })
    }, [scope])

    useEffect(() => {
        load()
    }, [load])

    useEffect(() => {
        if (!pendingOpenId) {
            return
        }
        setSelectedId(pendingOpenId)
        onPendingOpenHandled?.()
    }, [pendingOpenId, onPendingOpenHandled])

    if (selectedId) {
        return (
            <TicketDetail
                key={selectedId}
                scope={scope}
                ticketId={selectedId}
                onBack={() => setSelectedId(null)}
                onChanged={load}
            />
        )
    }

    return <TicketList tickets={tickets} error={error} onOpen={id => setSelectedId(id)} onRefresh={load} />
}
