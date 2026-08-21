import { useEffect, useMemo, useState } from 'react'
import { Box, Button, Stack, Typography } from '@mui/material'
import { FormProvider, useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IUserShort } from '@/features/user/types/user'
import type { ITaskDTO } from '../../types/task'
import type { FormValues, Props } from './types'
import { useAppSelector } from '@/hooks/redux'
import { getIsManager } from '@/features/user/userSlice'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAvailableUsersQuery } from '@/features/user/usersApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useCreateTaskMutation } from '../../tasksApiSlice'
import { useUploadAttachmentMutation } from '../../modules/attachments/attachmentsApiSlice'
import { getRealm } from '@/features/realms/realmSlice'
import { CategoryAndSiteSection } from './CategoryAndSiteSection'
import { DescriptionSection } from './DescriptionSection'
import { AdvancedSettingsSection } from './AdvancedSettingsSection'

export const TaskCreateForm = ({ onSuccess, onCancel, embedded }: Props) => {
	const currentUserId = useAppSelector(state => state.user.id)
	const realm = useAppSelector(getRealm)
	const isManager = useAppSelector(getIsManager)

	const [createTask, { isLoading: isCreating }] = useCreateTaskMutation()
	const [uploadAttachment, { isLoading: isUploading }] = useUploadAttachmentMutation()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: groupsData } = useGetAllGroupsQuery()
	const { data: usersData } = useGetAvailableUsersQuery()
	const { data: sitesData } = useGetAllSitesQuery()

	const [files, setFiles] = useState<File[]>([])

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const groups = useMemo(() => groupsData?.data ?? [], [groupsData])
	const users = useMemo(() => usersData?.data ?? [], [usersData])
	const sites = useMemo(() => sitesData?.data ?? [], [sitesData])

	const methods = useForm<FormValues>({
		defaultValues: {
			title: '',
			description: '',
			priority: 'medium',
			categoryId: '',
			groupId: null,
			ownerId: null,
			assigneeId: null,
			siteId: '',
			dueDate: null,
		},
	})
	const { control, handleSubmit, reset, setValue } = methods

	const selectedCategoryId = useWatch({ control, name: 'categoryId' })
	const selectedGroupId = useWatch({ control, name: 'groupId' })

	useEffect(() => {
		const cat = categories.find(c => c.id === selectedCategoryId)
		if (cat) setValue('priority', cat.priority)
	}, [selectedCategoryId, categories, setValue])

	useEffect(() => {
		if (!isManager) return
		const cat = categories.find(c => c.id === selectedCategoryId)
		if (cat?.groupId) {
			setValue('groupId', cat.groupId)
		} else {
			setValue('groupId', null)
		}
	}, [selectedCategoryId, categories, isManager, setValue])

	useEffect(() => {
		if (!isManager) return
		if (!selectedGroupId) {
			setValue('assigneeId', null)
			return
		}
		const group = groups.find(g => g.id === selectedGroupId)
		setValue('assigneeId', group?.defaultAssigneeId ?? null)
	}, [selectedGroupId, groups, isManager, setValue])

	const category = categories.find(c => c.id === selectedCategoryId)
	const selectedGroup = groups.find(g => g.id === selectedGroupId)
	const assigneeOptions: IUserShort[] = selectedGroup?.members ?? []

	const isSaving = isCreating || isUploading

	const onSubmit = handleSubmit(async data => {
		if (!currentUserId) {
			toast.error('Пользователь не найден')
			return
		}
		if (!realm?.id) {
			toast.error('Область не выбрана')
			return
		}

		const dto: ITaskDTO = {
			id: null,
			title: data.title,
			description: data.description,
			status: 'open',
			priority: isManager ? data.priority : category?.priority || 'medium',
			realmId: realm.id,
			siteId: data.siteId,
			categoryId: data.categoryId,
			creatorId: currentUserId,
			ownerId: isManager ? data.ownerId || null : null,
			groupId: isManager ? data.groupId || null : category?.groupId || null,
			assigneeId: isManager ? data.assigneeId || null : null,
			managerId: null,
			dueDate: isManager ? data.dueDate || null : null,
			closedAt: null,
		}

		try {
			const result = await createTask(dto).unwrap()

			if (files.length > 0) {
				const results = await Promise.allSettled(
					files.map(file => uploadAttachment({ entityType: 'ticket', entityId: result.id, file }).unwrap()),
				)
				const failed = results.filter(r => r.status === 'rejected').length
				if (failed > 0) {
					toast.warning(`Заявка создана, но ${failed} из ${files.length} файлов не загрузились`, {
						autoClose: false,
					})
				} else {
					toast.success('Задача создана')
				}
			} else {
				toast.success('Задача создана')
			}

			reset()
			setFiles([])
			onSuccess?.()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при создании задачи', { autoClose: false })
		}
	})

	return (
		<Box sx={{ maxWidth: !embedded ? 720 : undefined, mx: 'auto' }}>
			{!embedded && (
				<Box sx={{ mb: 3 }}>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						Создание задачи
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.5 }}>
						Заполните форму для создания новой задачи
					</Typography>
				</Box>
			)}

			<FormProvider {...methods}>
				<Box component='form' onSubmit={onSubmit}>
					<Stack sx={{ gap: 3 }}>
						<CategoryAndSiteSection categories={categories} sites={sites} />
						<DescriptionSection files={files} onFilesChange={setFiles} />

						{isManager && (
							<AdvancedSettingsSection groups={groups} users={users} assigneeOptions={assigneeOptions} />
						)}

						<Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, pt: 1, pb: !embedded ? 2 : 0 }}>
							<Button
								type='button'
								variant='outlined'
								onClick={embedded ? onCancel : () => reset()}
								sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
							>
								{embedded ? 'Отмена' : 'Очистить'}
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={isSaving}
								sx={{ textTransform: 'none', px: 3 }}
							>
								{isSaving ? 'Создание...' : 'Создать заявку'}
							</Button>
						</Box>
					</Stack>
				</Box>
			</FormProvider>
		</Box>
	)
}
