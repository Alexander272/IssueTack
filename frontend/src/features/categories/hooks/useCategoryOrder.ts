import { useCallback, useMemo, useState } from 'react'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { ICategory } from '../types/category'
import { useReorderCategoriesMutation } from '../categoriesApiSlice'

export const useCategoryOrder = (categories: ICategory[]) => {
	const [reorder, { isLoading: isSaving }] = useReorderCategoriesMutation()

	const serverIds = useMemo(() => categories.map(c => c.id), [categories])
	const signature = serverIds.join(',')
	const [local, setLocal] = useState<{ signature: string; ids: string[] } | null>(null)

	const ids = local && local.signature === signature ? local.ids : serverIds
	const byId = useMemo(() => new Map<string, ICategory>(categories.map(c => [c.id, c])), [categories])
	const ordered = useMemo(() => ids.map(id => byId.get(id)).filter((c): c is ICategory => Boolean(c)), [ids, byId])
	const isDirty = ids.join(',') !== signature

	const commit = useCallback((nextIds: string[]) => setLocal({ signature, ids: nextIds }), [signature])
	const reset = useCallback(() => setLocal(null), [])

	const save = useCallback(async () => {
		try {
			await reorder({ categoryIds: ids }).unwrap()
			setLocal(null)
			toast.success('Порядок сохранён')
		} catch (error) {
			toast.error((error as IFetchError).data?.message || 'Не удалось сохранить порядок', { autoClose: false })
		}
	}, [reorder, ids])

	return { ordered, isDirty, isSaving, commit, reset, save }
}
