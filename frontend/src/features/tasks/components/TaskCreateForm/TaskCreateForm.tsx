import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react'
import { Box, Button, Stack, Typography } from '@mui/material'
import { FormProvider, useForm, useWatch } from 'react-hook-form'
import { toast } from 'react-toastify'
import { SaveIcon } from 'lucide-mui'

import type { IFetchError } from '@/app/types/error'
import type { ITaskDTO } from '../../types/task'
import type { FormValues, Props } from './types'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { useAppSelector } from '@/hooks/redux'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useCreateTaskMutation } from '../../tasksApiSlice'
import { useUploadAttachmentMutation } from '../../modules/attachments/attachmentsApiSlice'
import { getCurrentCapabilities, getIsManager, getUserId, getUserSiteId } from '@/features/user/userSlice'
import { getRealm } from '@/features/realms/realmSlice'
import { CategoryAndSiteSection } from './CategoryAndSiteSection'
import { DescriptionSection } from './DescriptionSection'
import { AdvancedSettingsSection } from './AdvancedSettingsSection'
import { CustomerSelectionSection } from './CustomerSelectionSection'
import { SubtasksCreationSection } from './SubtasksCreationSection'
import { clearDraft, hasDraftContent, loadDraft, saveDraft } from './draft'

// EMPTY_FORM — исходные значения формы. reset() обязан сбрасывать именно к ним,
// а не к defaultValues: последние могут быть черновиком из sessionStorage.
const EMPTY_FORM: FormValues = {
	title: '',
	description: '',
	priority: 'medium',
	categoryId: '',
	groupId: null,
	ownerId: null,
	assigneeId: null,
	siteId: '',
	dueDate: null,
	subtasks: [],
}

export type TaskCreateFormHandle = { requestReset: () => void }

export const TaskCreateForm = forwardRef<TaskCreateFormHandle, Props>(
	({ onSuccess, onCancel, embedded, onSavingChange, onDirtyChange }, ref) => {
	const currentUserId = useAppSelector(getUserId)
	const realm = useAppSelector(getRealm)
	const isManager = useAppSelector(getIsManager)
	const capabilities = useAppSelector(getCurrentCapabilities)
	const userSiteId = useAppSelector(getUserSiteId)
	const isExecutor = !isManager && capabilities.memberGroupIds.length > 0

	const [createTask, { isLoading: isCreating }] = useCreateTaskMutation()
	const [uploadAttachment, { isLoading: isUploading }] = useUploadAttachmentMutation()
	const { data: categoriesData } = useGetAllCategoriesQuery()
	const { data: sitesData } = useGetAllSitesQuery()

	const [files, setFiles] = useState<File[]>([])
	const [submitting, setSubmitting] = useState(false)
	const [confirmOpen, setConfirmOpen] = useState(false)

	const categories = useMemo(() => categoriesData?.data ?? [], [categoriesData])
	const sites = useMemo(() => sitesData?.data ?? [], [sitesData])

	// Черновик читается синхронно на первом рендере, чтобы сразу попасть в
	// defaultValues формы. Ключ — realm + пользователь.
	const [initialDraft] = useState(() =>
		realm?.id && currentUserId ? loadDraft(realm.id, currentUserId) : null,
	)

	const methods = useForm<FormValues>({
		defaultValues: initialDraft ?? EMPTY_FORM,
		mode: 'onTouched',
	})
	const { control, getValues, handleSubmit, reset, setValue, formState } = methods

	const filesRef = useRef<File[]>([])
	const watchedValues = useWatch({ control }) as FormValues
	// Первый прогон эффекта сохранения пропускаем: форма уже инициализирована
	// черновиком, а частично пустой результат первого useWatch мог бы затереть
	// его в хранилище.
	const hydratedRef = useRef(false)

	useEffect(() => {
		filesRef.current = files
		onDirtyChange?.(hasDraftContent(getValues(), files.length))
	}, [files, filesRef, getValues, onDirtyChange])

	useEffect(() => {
		if (!realm?.id || !currentUserId) return
		if (!hydratedRef.current) {
			hydratedRef.current = true
			return
		}
		saveDraft(realm.id, currentUserId, watchedValues)
		onDirtyChange?.(hasDraftContent(watchedValues, filesRef.current.length))
	}, [realm?.id, currentUserId, watchedValues, onDirtyChange])

	const clearDraftStorage = () => {
		if (realm?.id && currentUserId) clearDraft(realm.id, currentUserId)
	}

	// Кнопка очистки живёт в шапке модалки (TaskCreateModal), поэтому наружу
	// отдаём только запрос на подтверждение — сам сброс делает форма.
	useImperativeHandle(ref, () => ({ requestReset: () => setConfirmOpen(true) }), [])

	const resetAll = useCallback(() => {
		reset(EMPTY_FORM)
		setFiles([])
		clearDraftStorage()
		setConfirmOpen(false)
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [reset, realm?.id, currentUserId])

	const selectedCategoryId = useWatch({ control, name: 'categoryId' })

	useEffect(() => {
		const cat = categories.find(c => c.id === selectedCategoryId)
		if (cat) setValue('priority', cat.priority, { shouldValidate: true })
	}, [selectedCategoryId, categories, setValue])

	useEffect(() => {
		if (userSiteId && !getValues('siteId')) {
			setValue('siteId', userSiteId, { shouldValidate: true })
		}
	}, [userSiteId, setValue, getValues])

	const category = categories.find(c => c.id === selectedCategoryId)

	const isSaving = isCreating || isUploading

	useEffect(() => {
		onSavingChange?.(isSaving)
	}, [isSaving, onSavingChange])

	const onSubmit = handleSubmit(async data => {
		if (submitting) return
		setSubmitting(true)
		try {
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
				title: data.title.trim(),
				description: data.description,
				status: 'open',
				priority: isManager ? data.priority : category?.priority || 'medium',
				realmId: realm.id,
				siteId: data.siteId,
				categoryId: data.categoryId,
				creatorId: currentUserId,
				ownerId: isManager || isExecutor ? data.ownerId || null : null,
				groupId: isManager ? data.groupId || null : category?.groupId || null,
				assigneeId: isManager || isExecutor ? data.assigneeId || null : null,
				managerId: null,
				dueDate: isManager ? data.dueDate || null : null,
				closedAt: null,
				subtasks: data.subtasks
					.map(s => ({ title: s.title.trim(), description: s.description.trim() }))
					.filter(s => s.title.length > 0),
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

				reset(EMPTY_FORM)
				setFiles([])
				clearDraftStorage()
				onSuccess?.()
			} catch (error) {
				const fetchError = error as IFetchError
				toast.error(fetchError.data?.message || 'Ошибка при создании задачи', { autoClose: false })
			}
		} finally {
			setSubmitting(false)
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
					<Stack sx={{ gap: { xs: 1.5, sm: 3 } }}>
						<CategoryAndSiteSection categories={categories} sites={sites} />
						<DescriptionSection files={files} onFilesChange={setFiles} />

						{(isManager || isExecutor) && <SubtasksCreationSection />}

						{isExecutor && <CustomerSelectionSection currentUserId={currentUserId} />}

						{isManager && <AdvancedSettingsSection />}

						<Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, pt: 1, pb: !embedded ? 2 : 0 }}>
							<Button
								type='button'
								variant='outlined'
								onClick={embedded ? onCancel : () => setConfirmOpen(true)}
								sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
							>
								{embedded ? 'Отмена' : 'Очистить'}
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={isSaving || !formState.isValid}
								startIcon={<SaveIcon sx={{ fontSize: 18 }} />}
								sx={{ textTransform: 'none', px: 3 }}
							>
								{isSaving ? 'Создание...' : 'Создать заявку'}
							</Button>
						</Box>
					</Stack>
				</Box>
			</FormProvider>

			<ConfirmDialog
				open={confirmOpen}
				title='Очистить черновик?'
				message='Заполненные данные и вложения будут потеряны.'
				confirmLabel='Очистить'
				confirmColor='warning'
				onConfirm={resetAll}
				onCancel={() => setConfirmOpen(false)}
			/>
		</Box>
	)
	},
)

TaskCreateForm.displayName = 'TaskCreateForm'
