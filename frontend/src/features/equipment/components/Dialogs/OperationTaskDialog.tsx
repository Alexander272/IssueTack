import { type FC } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	Divider,
	IconButton,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, FormProvider, useForm } from 'react-hook-form'
import { InfoIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type {
	EquipmentOperation,
	IEquipmentOperationTaskDTO,
} from '../../types/equipment'
import type { Priority } from '@/features/tasks/types/task'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { CategoryField } from '@/components/FormFields'
import { Switch } from '@/components/Switch/Switch'
import { useAppSelector } from '@/hooks/redux'
import { getRealm } from '@/features/realms/realmSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import {
	useGetOperationTaskQuery,
	useCreateOperationTaskMutation,
	useUpdateOperationTaskMutation,
} from '../../equipmentApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	taskId: string | null
}

const OPERATION_OPTIONS: { value: EquipmentOperation; label: string }[] = [
	{ value: 'assign', label: 'Выдача' },
	{ value: 'unassign', label: 'Возврат' },
]

const EMPTY_FORM: IEquipmentOperationTaskDTO = {
	realmId: '',
	name: '',
	operation: 'assign',
	action: 'create_ticket',
	title: '',
	description: '',
	categoryId: '',
	groupId: '',
	priority: 'medium',
	enabled: true,
}

// Диалог правила автоматической задачи. Условия срабатывания и параметры заявки
// задаются здесь; площадка заявки не показывается — сервер берёт её у единицы.
export const OperationTaskDialog: FC<Props> = ({ open, onClose, taskId }) => {
	const realm = useAppSelector(getRealm)
	const isEditing = taskId !== null && taskId !== ''

	const { data: existing } = useGetOperationTaskQuery(taskId ?? '', { skip: !taskId })
	const { data: groupsData } = useGetAllGroupsQuery()
	const groups = groupsData?.data ?? []

	const [createTask] = useCreateOperationTaskMutation()
	const [updateTask] = useUpdateOperationTaskMutation()

	const methods = useForm<IEquipmentOperationTaskDTO>({
		mode: 'onChange',
		defaultValues: EMPTY_FORM,
		values: isEditing && existing?.data
			? {
					id: existing.data.id,
					realmId: existing.data.realmId,
					name: existing.data.name,
					operation: existing.data.operation,
					action: existing.data.action,
					title: existing.data.title,
					description: existing.data.description,
					categoryId: existing.data.categoryId,
					groupId: existing.data.groupId,
					priority: existing.data.priority,
					enabled: existing.data.enabled,
				}
			: { ...EMPTY_FORM, realmId: realm?.id ?? '' },
	})
	const { handleSubmit, formState } = methods

	const onSubmit = async (data: IEquipmentOperationTaskDTO) => {
		const payload: IEquipmentOperationTaskDTO = {
			...data,
			action: 'create_ticket',
			title: data.title?.trim() || undefined,
			description: data.description?.trim() || undefined,
		}
		try {
			if (isEditing && data.id) {
				await updateTask(payload).unwrap()
				toast.success('Правило обновлено')
			} else {
				await createTask(payload).unwrap()
				toast.success('Правило создано')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при сохранении', { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			maxWidth='sm'
			fullWidth
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '12px' } } } }}
		>
			<DialogTitle
				sx={{
					m: 0,
					px: { xs: 1.5, sm: 3 },
					py: 2,
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
				}}
			>
				<Box>
					<Typography variant='h6' sx={{ fontWeight: 700 }}>
						{isEditing ? 'Редактирование правила' : 'Новое правило автоматической задачи'}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
						Создаётся после выдачи или возврата оборудования
					</Typography>
				</Box>
				<IconButton onClick={onClose}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<FormProvider {...methods}>
				<Box component='form' onSubmit={handleSubmit(onSubmit)}>
					<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: 2 }}>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Название правила *
								</Typography>
								<Controller
									name='name'
									rules={{ required: 'Название обязательно' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											fullWidth
											size='small'
											error={!!fieldState.error}
											helperText={fieldState.error?.message}
											placeholder='Например: Передача в 1C:ERP'
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Операция *
								</Typography>
								<Controller
									name='operation'
									rules={{ required: 'Выберите операцию' }}
									render={({ field, fieldState }) => (
										<Autocomplete
											{...field}
											fullWidth
											size='small'
											options={OPERATION_OPTIONS}
											getOptionLabel={o => o.label}
											getOptionKey={o => o.value}
											value={OPERATION_OPTIONS.find(o => o.value === field.value) ?? null}
											onChange={(_e, val) => field.onChange(val?.value ?? 'assign')}
											renderInput={params => (
												<TextField
													{...params}
													error={Boolean(fieldState.error)}
													helperText={fieldState.error?.message}
												/>
											)}
										/>
									)}
								/>
							</Box>
						</Box>

						<Divider sx={{ borderColor: '#f3f4f6', my: 2 }} />

						<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
							Параметры создаваемой заявки
						</Typography>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mt: 1 }}>
							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Заголовок
								</Typography>
								<Controller
									name='title'
									render={({ field }) => (
										<TextField
											{...field}
											fullWidth
											size='small'
											placeholder='Провести передачу техники в 1C:ERP'
										/>
									)}
								/>
								<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
									Пустое поле — стандартный заголовок операции
								</Typography>
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Описание
								</Typography>
								<Controller
									name='description'
									render={({ field }) => (
										<TextField
											{...field}
											fullWidth
											size='small'
											multiline
											rows={2}
											placeholder='Стандартное описание операции'
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Категория *
								</Typography>
								<CategoryField name='categoryId' rules={{ required: 'Обязательное поле' }} />
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Группа *
								</Typography>
								<Controller
									name='groupId'
									rules={{ required: 'Выберите группу' }}
									render={({ field, fieldState }) => (
										<Autocomplete
											{...field}
											fullWidth
											size='small'
											options={groups}
											getOptionLabel={o => o.name}
											getOptionKey={o => o.id}
											value={groups.find(g => g.id === field.value) ?? null}
											onChange={(_e, val) => field.onChange(val?.id ?? '')}
											noOptionsText='Нет групп'
											renderInput={params => (
												<TextField
													{...params}
													error={Boolean(fieldState.error)}
													helperText={fieldState.error?.message}
												/>
											)}
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, color: '#374151' }}>
									Приоритет
								</Typography>
								<Controller
									name='priority'
									render={({ field }) => (
										<Autocomplete
											{...field}
											fullWidth
											size='small'
											options={Object.keys(PRIORITY_MAP) as Priority[]}
											getOptionLabel={(value: Priority) => PRIORITY_MAP[value].label}
											getOptionKey={(value: Priority) => value}
											value={field.value || null}
											onChange={(_e, val) => field.onChange(val || 'medium')}
											renderInput={params => <TextField {...params} />}
										/>
									)}
								/>
							</Box>
						</Box>

						<Box
							component='label'
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 2,
								mt: 2,
								py: 1,
								px: 2,
								border: '1px solid #e5e7eb',
								borderRadius: '8px',
								cursor: 'pointer',
								'&:has(input:checked)': { borderColor: '#60a5fa', bgcolor: '#eff6ff' },
							}}
						>
							<Controller
								name='enabled'
								render={({ field }) => <Switch {...field} checked={field.value} />}
							/>
							<Box>
								<Typography variant='subtitle2' sx={{ fontWeight: 500, color: '#111827' }}>
									Правило активно
								</Typography>
								<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
									Отключённое правило заявки не создаёт
								</Typography>
							</Box>
						</Box>
					</DialogContent>

					<DialogActions sx={{ px: { xs: 1.5, sm: 3 }, py: 2, justifyContent: 'space-between' }}>
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
							<InfoIcon sx={{ fontSize: 14, color: '#9ca3af' }} />
							<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
								Площадка заявки берётся у оборудования
							</Typography>
						</Box>
						<Box sx={{ display: 'flex', gap: 1 }}>
							<Button
								onClick={onClose}
								variant='outlined'
								sx={{ textTransform: 'none', borderColor: '#d1d5db', borderRadius: '8px' }}
							>
								Отмена
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={!formState.isValid}
								sx={{
									textTransform: 'none',
									px: 3,
									borderRadius: '8px',
									bgcolor: '#0432a5',
									'&:hover': { bgcolor: '#002d91' },
								}}
							>
								{isEditing ? 'Сохранить' : 'Создать'}
							</Button>
						</Box>
					</DialogActions>
				</Box>
			</FormProvider>
		</Dialog>
	)
}
