import React, { useEffect, useState } from 'react'

import type { PluginContextResult, PluginScope } from '../types'
import CreateTab from './CreateTab'
import MyTicketsTab from './MyTicketsTab'
import { CloseIcon } from './icons'

interface ModalControllerProps {
	scope: PluginScope
	context: PluginContextResult
	onClose: () => void
}

type TabId = 'create' | 'mine'

const TABS: { id: TabId; label: string }[] = [
	{ id: 'create', label: 'Создать заявку' },
	{ id: 'mine', label: 'Мои заявки' },
]

export default function ModalController({ scope, context, onClose }: ModalControllerProps) {
	const [tab, setTab] = useState<TabId>('create')
	// Заявка, которую нужно открыть сразу после переключения на «Мои заявки»:
	// её id приходит с экрана успеха CreateTab. Хранится здесь, а не в MyTicketsTab,
	// чтобы источник открытия (создание или клик в списке) не зависел от внутреннего
	// состояния вкладки.
	const [pendingOpenId, setPendingOpenId] = useState<string | null>(null)

	useEffect(() => {
		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape') {
				onClose()
			}
		}
		window.addEventListener('keydown', onKey)
		return () => window.removeEventListener('keydown', onKey)
	}, [onClose])

	return (
		<div className='it-ticket-overlay' onClick={onClose}>
			<div
				className='it-ticket-modal'
				role='dialog'
				aria-modal='true'
				aria-label='IssueTrack'
				onClick={e => e.stopPropagation()}
			>
				<div className='it-ticket-modal__header'>
					<div className='it-ticket-modal__title'>Заявки в IT отдел</div>
					<button type='button' className='it-ticket-modal__close' aria-label='Закрыть' onClick={onClose}>
						<CloseIcon size={16} />
					</button>
				</div>
				<div className='it-ticket-modal__tabs'>
					{TABS.map(t => (
						<button
							key={t.id}
							type='button'
							className={`it-ticket-tab${tab === t.id ? ' it-ticket-tab--active' : ''}`}
							onClick={() => setTab(t.id)}
						>
							{t.label}
						</button>
					))}
				</div>
				<div className='it-ticket-modal__body'>
					{tab === 'create' ? (
						<CreateTab
							scope={scope}
							context={context}
							onCreated={() => setTab('mine')}
							onOpen={id => {
								setTab('mine')
								setPendingOpenId(id)
							}}
						/>
					) : (
						<MyTicketsTab
							scope={scope}
							pendingOpenId={pendingOpenId}
							onPendingOpenHandled={() => setPendingOpenId(null)}
						/>
					)}
				</div>
			</div>
		</div>
	)
}
