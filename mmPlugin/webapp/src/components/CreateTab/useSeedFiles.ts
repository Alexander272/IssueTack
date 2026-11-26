// Вложения из сообщения Mattermost качаем браузером с его сессией: у bot-тока
// может не быть доступа к файлам канала. Ограничения те же, что у бэкенда;
// пропущенное не пропадает молча — причина показывается в форме.

import {useEffect, useState, type Dispatch, type SetStateAction} from 'react'

import {getMattermostFile} from '../../api'
import type {CreateTicketSeed, PluginPostFileRef} from '../../types'
import {MAX_SEED_FILE_BYTES, MAX_SEED_FILES} from './model'

export function useSeedFiles(initialSeed?: CreateTicketSeed | null): {
	files: File[]
	setFiles: Dispatch<SetStateAction<File[]>>
	seedWarning: string | null
} {
	const [files, setFiles] = useState<File[]>([])
	const [seedWarning, setSeedWarning] = useState<string | null>(null)

	useEffect(() => {
		if (!initialSeed?.files?.length) return
		let cancelled = false
		const accepted: PluginPostFileRef[] = []
		let total = 0
		let tooMany = false
		let tooLarge = false
		for (const ref of initialSeed.files) {
			if (accepted.length >= MAX_SEED_FILES) {
				tooMany = true
				break
			}
			const size = typeof ref.size === 'number' ? ref.size : 0
			if (total + size > MAX_SEED_FILE_BYTES) {
				tooLarge = true
				continue
			}
			accepted.push(ref)
			total += size
		}
		Promise.all(accepted.map(ref => getMattermostFile(ref).catch(() => null)))
			.then(results => {
				if (cancelled) return
				const loaded = results.filter((f): f is File => Boolean(f))
				if (loaded.length) {
					setFiles(prev => [...prev, ...loaded].slice(0, MAX_SEED_FILES))
				}
				const failed = accepted.length - loaded.length
				const notes: string[] = []
				if (tooMany) notes.push(`перенесено только ${MAX_SEED_FILES} вложений`)
				if (tooLarge) notes.push('часть файлов превышает лимит 64 МБ')
				if (failed) notes.push(`не удалось скачать ${failed} файл(ов)`)
				if (notes.length) setSeedWarning(notes.join('; '))
			})
		return () => {
			cancelled = true
		}
	}, [initialSeed])

	return {files, setFiles, seedWarning}
}
