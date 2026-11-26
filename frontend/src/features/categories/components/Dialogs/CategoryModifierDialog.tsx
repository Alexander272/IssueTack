import { useId, useState, type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormLabel,
	IconButton,
	MenuItem,
	Stack,
	TextField,
	Tooltip,
	Typography,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { toast } from 'react-toastify'
import { CircleQuestionMarkIcon, XIcon } from 'lucide-mui'

import type { IFetchError } from '@/app/types/error'
import type { ICategory, ICategoryModifierDTO } from '../../types/category'
import type { Priority } from '@/features/tasks/types/task'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import {
	useCreateCategoryModifierMutation,
	useUpdateCategoryModifierMutation,
	useDeleteCategoryModifierMutation,
} from '../../categoriesApiSlice'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import { Switch } from '@/components/Switch/Switch'
import { KeywordsField } from '@/components/FormFields/KeywordsField'

type Props = {
	modifier?: ICategoryModifierDTO
	categories: ICategory[]
	open: boolean
	onClose: () => void
}

const emptyModifier = (categoryId: string): ICategoryModifierDTO => ({
	id: null,
	categoryId,
	name: '',
	description: '',
	keywords: [],
	defaultDueDays: null,
	priority: null,
	isActive: true,
})

export const CategoryModifierDialog: FC<Props> = ({ modifier, categories, open, onClose }) => {
	const labelId = useId()
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [create, { isLoading: isCreating }] = useCreateCategoryModifierMutation()
	const [update, { isLoading: isUpdating }] = useUpdateCategoryModifierMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteCategoryModifierMutation()

	const { control, handleSubmit, formState } = useForm<ICategoryModifierDTO>({
		values: modifier ?? emptyModifier(categories[0]?.id ?? ''),
		mode: 'onTouched',
	})
	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async form => {
		try {
			if (form.id) {
				await update(form).unwrap()
				toast.success('Подкатегория обновлена')
			} else {
				await create(form).unwrap()
				toast.success('Подкатегория создана')
			}
			onClose()
		} catch (error) {
			toast.error((error as IFetchError).data?.message, { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!modifier?.id) return
		try {
			await remove(modifier.id).unwrap()
			toast.success('Подкатегория удалена')
			setDeleteOpen(false)
			onClose()
		} catch (error) {
			toast.error((error as IFetchError).data?.message, { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			maxWidth='md'
			slotProps={{ paper: { sx: { borderRadius: '16px', p: 1 } } }}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					{modifier?.id ? 'Редактировать подкатегорию' : 'Создать подкатегорию'}
				</Typography>
				<IconButton size='large' onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Stack sx={{ mb: 3 }}>
					<Typography sx={{ color: '#6b7280', mb: 1 }}>Основная информация</Typography>
					<Stack direction={'row'} spacing={2} sx={{ mb: 2 }}>
						<Controller
							control={control}
							name='categoryId'
							rules={{ required: 'Обязательное поле' }}
							render={({ field, fieldState }) => (
								<TextField
									select
									label='Категория'
									fullWidth
									value={field.value || ''}
									onChange={field.onChange}
									error={Boolean(fieldState.error)}
									helperText={fieldState.error?.message}
								>
									{categories.map(c => (
										<MenuItem key={c.id} value={c.id}>
											{c.name}
										</MenuItem>
									))}
								</TextField>
							)}
						/>

						<Controller
							control={control}
							name='name'
							rules={{ required: 'Обязательное поле' }}
							render={({ field, fieldState }) => (
								<TextField
									label='Название'
									fullWidth
									{...field}
									error={Boolean(fieldState.error)}
									helperText={fieldState.error?.message}
								/>
							)}
						/>
					</Stack>
					<Controller
						control={control}
						name='description'
						render={({ field }) => (
							<TextField label='Описание' fullWidth multiline minRows={2} {...field} />
						)}
					/>
				</Stack>

				<Stack sx={{ mb: 3 }}>
					<Typography sx={{ color: '#6b7280', mb: 1 }}>Автоматизация</Typography>

					<Stack sx={{ mb: 2 }}>
						<Controller
							control={control}
							name='keywords'
							render={({ field, fieldState }) => (
								<KeywordsField
									value={field.value}
									onChange={field.onChange}
									error={!!fieldState.error}
									helperText={fieldState.error?.message}
								/>
							)}
						/>
					</Stack>

					<Stack direction={'row'} spacing={2}>
						<Controller
							control={control}
							name='defaultDueDays'
							render={({ field }) => (
								<TextField
									label='Автоматический срок, рабочих дней'
									size='small'
									fullWidth
									type='number'
									value={field.value ?? ''}
									onChange={e => {
										const raw = e.target.value
										field.onChange(raw === '' ? null : Math.max(0, Number.parseInt(raw, 10) || 0))
									}}
									slotProps={{ htmlInput: { min: 0, step: 1 } }}
									helperText={
										<Box
											component='span'
											sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
										>
											Пусто — наследовать срок категории
											<Tooltip title='Выходные и праздничные дни не учитываются'>
												<CircleQuestionMarkIcon sx={{ fontSize: 14, cursor: 'pointer' }} />
											</Tooltip>
										</Box>
									}
								/>
							)}
						/>

						<Controller
							control={control}
							name='priority'
							render={({ field }) => (
								<TextField
									select
									label='Приоритет'
									size='small'
									fullWidth
									value={field.value ?? ''}
									onChange={e => field.onChange(e.target.value || null)}
									helperText='Пусто — наследовать приоритет категории'
								>
									<MenuItem value=''>По категории</MenuItem>
									{(Object.keys(PRIORITY_MAP) as Priority[]).map(value => (
										<MenuItem key={value} value={value}>
											{PRIORITY_MAP[value].label}
										</MenuItem>
									))}
								</TextField>
							)}
						/>
					</Stack>
				</Stack>

				<Stack spacing={1}>
					<Typography sx={{ color: '#6b7280' }}>Статус</Typography>
					<Controller
						control={control}
						name='isActive'
						render={({ field }) => (
							<Stack
								direction={'row'}
								sx={{
									border: '1px solid #eef0f3',
									background: '#f9fafb',
									py: 1,
									px: 2,
									borderRadius: 3,
									justifyContent: 'space-between',
									alignItems: 'center',
								}}
							>
								<Stack direction={'row'} spacing={2} sx={{ alignItems: 'center' }}>
									<Typography>{modifier?.isActive ? 'Активна' : 'Неактивна'}</Typography>

									<FormLabel
										htmlFor={labelId}
										sx={{
											color: field.value ? '#6b7280' : '#9ca3af',
											fontSize: 14,
											cursor: 'pointer',
										}}
									>
										{field.value ? '● Включено' : '○ Выключено'}
									</FormLabel>
								</Stack>

								<Switch
									id={labelId}
									checked={field.value}
									onChange={e => field.onChange(e.target.checked)}
								/>
							</Stack>
						)}
					/>
				</Stack>
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1, justifyContent: modifier?.id ? 'space-between' : 'flex-end' }}>
				{modifier?.id && (
					<Button
						type='button'
						variant='outlined'
						color='error'
						onClick={() => setDeleteOpen(true)}
						disabled={isDeleting}
						sx={{ textTransform: 'none', borderColor: '#fecaca', '&:hover': { borderColor: '#fca5a5' } }}
					>
						Удалить
					</Button>
				)}
				<Box sx={{ display: 'flex', gap: 1 }}>
					<Button
						onClick={onClose}
						variant='outlined'
						sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
					>
						Отмена
					</Button>
					<Button
						onClick={saveHandler}
						variant='contained'
						disabled={isLoading || !formState.isValid || (Boolean(modifier?.id) && !formState.isDirty)}
						sx={{ textTransform: 'none', px: 3 }}
					>
						{modifier?.id ? 'Сохранить' : 'Создать'}
					</Button>
				</Box>
			</DialogActions>

			<ConfirmDialog
				open={deleteOpen}
				title='Удаление подкатегории'
				message={`Удалить подкатегорию «${modifier?.name}»?`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</Dialog>
	)
}
