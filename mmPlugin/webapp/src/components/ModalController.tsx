import React, { useEffect, useState } from 'react'

import type { PluginContextResult, PluginScope } from '../types'
import CreateTab from './CreateTab'
import MyTicketsTab from './MyTicketsTab'
import { CloseIcon } from './icons'

interface ModalControllerProps {
	scope: PluginScope
	context: PluginContextResult
	// Заявка, которую нужно открыть сразу при монтировании (ссылка из DM, см.
	// deepLink.ts). Отдельный проп, а не onOpen, потому что сценарий открытия
	// уже завершён: вкладка должна быть сразу «Мои заявки». На лету не меняется:
	// при открытой модалке чат перекрыт подложкой, нажать вторую ссылку нельзя.
	initialPendingOpenId?: string | null
	onClose: () => void
}

type TabId = 'create' | 'mine'

const TABS: { id: TabId; label: string }[] = [
	{ id: 'create', label: 'Создать заявку' },
	{ id: 'mine', label: 'Мои заявки' },
]

export default function ModalController({scope, context, initialPendingOpenId, onClose}: ModalControllerProps) {
	const [tab, setTab] = useState<TabId>(initialPendingOpenId ? 'mine' : 'create')
	// Заявка, которую нужно открыть сразу после переключения на «Мои заявки»:
	// её id приходит с экрана успеха CreateTab. Хранится здесь, а не в MyTicketsTab,
	// чтобы источник открытия (создание или клик в списке) не зависел от внутреннего
	// состояния вкладки.
	const [pendingOpenId, setPendingOpenId] = useState<string | null>(initialPendingOpenId ?? null)
	// Есть ли непустой черновик формы создания: пока он есть, клик мимо окна и
	// Escape не закрывают модалку (закрыть можно только крестиком/по действию) —
	// иначе заполненная форма терялась бы от случайного клика. Пустая форма
	// закрывается как раньше.
	const [hasDraft, setHasDraft] = useState(false)

	useEffect(() => {
		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape' && !hasDraft) {
				onClose()
			}
		}
		window.addEventListener('keydown', onKey)
		return () => window.removeEventListener('keydown', onKey)
	}, [onClose, hasDraft])

	return (
		<div className='it-ticket-overlay' onClick={() => (hasDraft ? undefined : onClose())}>
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
							onDraftChange={setHasDraft}
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
