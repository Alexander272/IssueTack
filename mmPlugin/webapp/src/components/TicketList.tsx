import React from 'react'

import type { PluginTicketShort } from '../types'
import { formatDate, statusMeta } from '../labels'
import { EyeIcon } from './icons'

interface TicketListProps {
    tickets: PluginTicketShort[] | null
    error: string | null
    onOpen: (id: string) => void
    onRefresh: () => void
}

export default function TicketList({ tickets, error, onOpen, onRefresh }: TicketListProps) {
    const sorted = (tickets || []).slice().sort((a, b) => {
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    })

    return (
        <div className='it-ticket-list'>
            <div className='it-ticket-list__toolbar'>
                <span className='it-ticket-list__count'>{tickets ? `Мои заявки: ${tickets.length}` : 'Загрузка…'}</span>
                <button type='button' className='it-btn it-btn--sm' onClick={onRefresh}>
                    Обновить
                </button>
            </div>

            {error ? <div className='it-ticket-error'>{error}</div> : null}

            {!error && tickets && sorted.length === 0 ? (
                <div className='it-ticket-list__empty'>Заявок пока нет.</div>
            ) : null}

            {tickets ? (
                <ul className='it-ticket-items'>
                    {sorted.map(t => {
                        const status = statusMeta(t.status)
                        return (
                            <li key={t.id} className='it-ticket-card'>
                                <div
                                    className='it-ticket-card__link'
                                    role='button'
                                    tabIndex={0}
                                    onClick={() => onOpen(t.id)}
                                    onKeyDown={e => {
                                        if (e.key === 'Enter' || e.key === ' ') {
                                            e.preventDefault()
                                            onOpen(t.id)
                                        }
                                    }}
                                >
                                    <div className='it-ticket-card__body'>
                                        <div className='it-ticket-card__head'>
                                            {t.number ? <span className='it-ticket-card__num'>№{t.number}</span> : null}
                                            <span className='it-ticket-card__title'>{t.title}</span>
                                        </div>
                                        {t.description ? (
                                            <div className='it-ticket-card__desc'>{t.description}</div>
                                        ) : null}
                                        <div className='it-ticket-card__chips'>
                                            <span
                                                className='it-badge it-badge--status'
                                                style={{ background: status.bg, color: status.text }}
                                            >
                                                {status.label}
                                            </span>
                                            {t.site ? <span className='it-chip'>{t.site.name}</span> : null}
                                        </div>
                                        <div className='it-ticket-card__foot'>
                                            <span className='it-ticket-card__date'>
                                                Создана: {formatDate(t.createdAt)}
                                            </span>
                                            <span
                                                className='it-ticket-card__open'
                                                title='Открыть заявку'
                                                onClick={() => onOpen(t.id)}
                                            >
                                                <EyeIcon size={16} />
                                                <span>Открыть</span>
                                            </span>
                                        </div>
                                    </div>
                                </div>
                            </li>
                        )
                    })}
                </ul>
            ) : null}
        </div>
    )
}