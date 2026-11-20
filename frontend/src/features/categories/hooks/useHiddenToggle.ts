import { useCallback } from 'react'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ICategory } from '../types/category'
import type { ICategoryGroup } from '../types/categoryGroup'
import { useUpdateCategoryMutation } from '../categoriesApiSlice'
import { useUpdateCategoryGroupMutation } from '../categoryGroupsApiSlice'
import { toCategoryDTO, toGroupDTO } from '../utils/dto'

export const useHiddenToggle = () => {
	const [updateCategory] = useUpdateCategoryMutation()
	const [updateGroup] = useUpdateCategoryGroupMutation()

	const toggleCategoryHidden = useCallback(
		async (cat: ICategory) => {
			try {
				await updateCategory({ ...toCategoryDTO(cat), isHidden: !cat.isHidden }).unwrap()
				toast.success(cat.isHidden ? 'Категория снова видна в списках' : 'Категория скрыта из списков выбора')
			} catch (error) {
				toast.error((error as IFetchError).data?.message, { autoClose: false })
			}
		},
		[updateCategory],
	)

	const toggleSectionHidden = useCallback(
		async (section: ICategoryGroup) => {
			try {
				await updateGroup({ ...toGroupDTO(section), isHidden: !section.isHidden }).unwrap()
				toast.success(section.isHidden ? 'Раздел снова виден в списках' : 'Раздел скрыт из списков выбора')
			} catch (error) {
				toast.error((error as IFetchError).data?.message, { autoClose: false })
			}
		},
		[updateGroup],
	)

	return { toggleCategoryHidden, toggleSectionHidden }
}
