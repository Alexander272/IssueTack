import { useCallback, useState } from 'react'

import type { ICategory } from '../types/category'
import type { IGroup } from '@/features/groups/types/group'
import type { CategoryDialogRequest, CategoryDialogState } from '../types/dialogs'
import { toCategoryDTO, toGroupDTO, toModifierDTO } from '../utils/dto'

export const useCategoryDialogs = ({ groups, categories }: { groups: IGroup[]; categories: ICategory[] }) => {
	// Открыт максимум один диалог — одно состояние-union вместо трёх.
	const [dialog, setDialog] = useState<CategoryDialogState>(null)

	const openDialog = useCallback(
		(request: CategoryDialogRequest) => {
			switch (request.kind) {
				case 'create-category':
					return setDialog({
						kind: 'category',
						data: {
							id: null,
							name: '',
							description: '',
							groupId: groups[0]?.id || '',
							categoryGroupId: request.sectionId ?? null,
							priority: 'medium',
							isActive: true,
							isHidden: false,
							defaultDueDays: null,
						},
					})
				case 'edit-category':
					return setDialog({ kind: 'category', data: toCategoryDTO(request.category) })
				case 'create-section':
					return setDialog({
						kind: 'section',
						data: { id: null, name: '', description: '', isHidden: false },
					})
				case 'edit-section':
					return setDialog({ kind: 'section', data: toGroupDTO(request.section) })
				case 'create-modifier':
					return setDialog({
						kind: 'modifier',
						data: {
							id: null,
							categoryId: request.categoryId || categories[0]?.id || '',
							name: '',
							description: '',
							defaultDueDays: null,
							priority: null,
							isActive: true,
						},
					})
				case 'edit-modifier':
					return setDialog({ kind: 'modifier', data: toModifierDTO(request.modifier) })
			}
		},
		[groups, categories],
	)

	const closeDialog = useCallback(() => setDialog(null), [])

	return { dialog, openDialog, closeDialog }
}
