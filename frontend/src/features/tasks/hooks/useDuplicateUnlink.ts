import { useState } from 'react'
import { toast } from 'react-toastify'

import { useUnlinkDuplicateMutation } from '../tasksApiSlice'

/** Цель отвязки: дубль из списка либо сама открытая заявка. */
export interface DuplicateUnlinkTarget {
	id: string
	ticketNumber?: number
	isSelf?: boolean
}

/**
 * Общее состояние отвязки для баннера и строк списка дублей: диалог
 * подтверждения, мутация и тост. Одновременно они не рендерятся (баннер
 * показывается вместо карточки с дулями), но логика у них одинаковая.
 */
export const useDuplicateUnlink = () => {
	const [target, setTarget] = useState<DuplicateUnlinkTarget | null>(null)
	const [unlinkDuplicate, { isLoading: unlinking }] = useUnlinkDuplicateMutation()

	const confirm = async () => {
		if (!target) return
		try {
			await unlinkDuplicate(target.id).unwrap()
			toast.success(
				target.isSelf
					? 'Дубль отвязан, заявка возвращена в работу'
					: `Связь с дублем №${target.ticketNumber ?? '—'} удалена`,
			)
			setTarget(null)
		} catch {
			// ошибка уже показана тостом в tasksApiSlice
		}
	}

	return { target, setTarget, unlinking, confirm }
}
