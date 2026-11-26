// Предпросмотр авто-подбора подкатегории: сервер выберет ту же подкатегорию при
// сохранении. Пока пользователь не кликнул по подсказке, поле остаётся пустым.
// Запрос уходит только тем, кто может выбирать подкатегорию вручную, и только
// при непустом тексте; устаревшие ответы гасятся.

import {useCallback, useEffect, useMemo, useState} from 'react'

import {previewModifier} from '../../api'
import type {PluginModifierShort, PluginScope} from '../../types'
import {useDebounce} from '../../useDebounce'

interface ModifierPreviewParams {
	scope: PluginScope
	categoryId: string
	title: string
	description: string
	// enabled — права и наличие категории/подкатегорий; уже выбранную
	// подкатегорию проверяет родитель, чтобы не дергать сервер после клика.
	enabled: boolean
	selectedId: string
	onSelect: (id: string) => void
}

export function useModifierPreview({
	scope,
	categoryId,
	title,
	description,
	enabled,
	selectedId,
	onSelect,
}: ModifierPreviewParams) {
	const [preview, setPreview] = useState<PluginModifierShort | null>(null)

	// Scope мемоизируется по его полям: объект из пропсов может меняться при
	// ре-рендере и вызывать лишние запросы.
	const previewScope = useMemo<PluginScope>(
		() => ({channelId: scope.channelId, userId: scope.userId, botUserId: scope.botUserId}),
		[scope.channelId, scope.userId, scope.botUserId],
	)
	const debouncedTitle = useDebounce(title, 350)
	const debouncedDescription = useDebounce(description, 350)
	const canPreview =
		enabled &&
		!selectedId &&
		Boolean(debouncedTitle.trim() || debouncedDescription.trim())

	useEffect(() => {
		if (!canPreview) {
			setPreview(null)
			return
		}
		let actual = true
		previewModifier(previewScope, {
			categoryId,
			title: debouncedTitle,
			description: debouncedDescription,
		})
			.then(found => {
				if (actual) setPreview(found)
			})
			.catch(() => {
				// Подсказка — удобство, её сбой не должен мешать созданию заявки.
				if (actual) setPreview(null)
			})
		return () => {
			actual = false
		}
	}, [canPreview, categoryId, debouncedTitle, debouncedDescription, previewScope])

	const showPreview = canPreview && Boolean(preview)
	const reset = useCallback(() => setPreview(null), [])
	const applyPreview = useCallback(() => {
		if (preview) onSelect(preview.id)
	}, [preview, onSelect])

	return {showPreview, preview, reset, applyPreview}
}
