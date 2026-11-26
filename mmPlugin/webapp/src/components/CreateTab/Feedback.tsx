// Экраны-состояния формы creation: результат после успешного создания и
// подтверждение очистки черновика. Не входят в основную форму, поэтому вынесены
// отдельно от полей.

import React from 'react'

import type {PluginCreateResult} from '../../types'

interface SuccessViewProps {
	result: PluginCreateResult
	onOpen: (ticketId: string) => void
	onCreated: () => void
	onReset: () => void
}

export function SuccessView({result, onOpen, onCreated, onReset}: SuccessViewProps) {
	return (
		<div className='it-ticket-success'>
			<div className='it-ticket-success__title'>Заявка №{result.number || ''} создана</div>
			<div className='it-ticket-success__actions'>
				<button type='button' className='it-btn it-btn--primary' onClick={() => onOpen(result.id)}>
					Открыть заявку
				</button>
				<button type='button' className='it-btn' onClick={onCreated}>
					К моим заявкам
				</button>
				<button type='button' className='it-btn' onClick={onReset}>
					Создать ещё
				</button>
			</div>
			{result.link ? (
				<a className='it-ticket-success__link' href={result.link} target='_blank' rel='noreferrer'>
					Открыть в веб-приложении
				</a>
			) : null}
		</div>
	)
}

interface ClearConfirmDialogProps {
	onCancel: () => void
	onConfirm: () => void
}

export function ClearConfirmDialog({onCancel, onConfirm}: ClearConfirmDialogProps) {
	return (
		<div className='it-confirm' onClick={onCancel}>
			<div className='it-confirm__box' onClick={e => e.stopPropagation()}>
				<div className='it-confirm__title'>Очистить черновик?</div>
				<div className='it-confirm__text'>Заполненные данные и вложения будут потеряны.</div>
				<div className='it-confirm__actions'>
					<button type='button' className='it-btn it-btn--sm' onClick={onCancel}>
						Нет
					</button>
					<button
						type='button'
						className='it-btn it-btn--sm it-btn--danger it-btn--solid'
						onClick={onConfirm}
					>
						Очистить
					</button>
				</div>
			</div>
		</div>
	)
}
