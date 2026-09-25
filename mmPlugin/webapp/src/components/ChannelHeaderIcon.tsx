import React, { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'

import { getContext, getContextFresh, getCachedContext, getCurrentUserId } from '../api'
import type { PluginContextResult } from '../types'
import ModalController from './ModalController'
import { ClipboardCheckIcon } from './icons'

interface ChannelHeaderIconProps {
	channel: { id: string }
}

export default function ChannelHeaderIcon({ channel }: ChannelHeaderIconProps) {
	const channelId = channel ? channel.id : null
	const userId = getCurrentUserId()
	const [context, setContext] = useState<PluginContextResult | null>(() => getCachedContext(channelId, userId))
	const [ready, setReady] = useState<boolean>(() => !!getCachedContext(channelId, userId))
	const [open, setOpen] = useState(false)

	useEffect(() => {
		if (!channelId || !userId) {
			return undefined
		}

		const fromCache = getCachedContext(channelId, userId)
		if (fromCache) {
			setContext(fromCache)
			setReady(true)
			return undefined
		}

		let cancelled = false
		setReady(false)
		getContext(channelId, userId)
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
	}, [channelId, userId])

	if (!ready || !context || !context.bound) {
		return null
	}

	const openModal = () => {
		setOpen(true)
		if (!channelId || !userId) {
			return
		}
		getContextFresh(channelId, userId)
			.then(data => setContext(data))
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
			{open && channelId && userId
				? createPortal(
						<ModalController
							channelId={channelId}
							userId={userId}
							context={context}
							onClose={() => setOpen(false)}
						/>,
						document.body,
					)
				: null}
		</>
	)
}
