import React, { useCallback, useEffect, useState } from 'react'

import { ApiError, getMyTickets } from '../api'
import type { PluginTicketShort } from '../types'
import TicketList from './TicketList'
import TicketDetail from './TicketDetail'

interface MyTicketsTabProps {
    channelId: string
    userId: string
    onOpen: () => void
}

export default function MyTicketsTab({ channelId, userId }: MyTicketsTabProps) {
    const [tickets, setTickets] = useState<PluginTicketShort[] | null>(null)
    const [error, setError] = useState<string | null>(null)
    const [selectedId, setSelectedId] = useState<string | null>(null)

    const load = useCallback(() => {
        setTickets(null)
        setError(null)
        getMyTickets(channelId, userId)
            .then(list => setTickets(list || []))
            .catch((err: unknown) => setError(err instanceof ApiError ? err.message : 'Не удалось загрузить заявки'))
    }, [channelId, userId])

    useEffect(() => {
        load()
    }, [load])

    if (selectedId) {
        return (
            <TicketDetail
                key={selectedId}
                channelId={channelId}
                userId={userId}
                ticketId={selectedId}
                onBack={() => setSelectedId(null)}
                onChanged={load}
            />
        )
    }

    return <TicketList tickets={tickets} error={error} onOpen={id => setSelectedId(id)} onRefresh={load} />
}