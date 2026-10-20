import { type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormHelperText,
	IconButton,
	MenuItem,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useFieldArray, useForm } from 'react-hook-form'
import { PlusIcon, Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentKitTemplate, IEquipmentKitTemplateDTO } from '../../types/equipment'
import {
	useCreateEquipmentTemplateMutation,
	useGetAllEquipmentTypesQuery,
	useUpdateEquipmentTemplateMutation,
} from '../../equipmentApiSlice'
import { Field } from './UnitForm/Field'

type Props = {
	open: boolean
	onClose: () => void
	template?: IEquipmentKitTemplate | null
}

type FormValues = {
	name: string
	description: string
	items: { typeId: string }[]
}

// Диалог создания/правки шаблона комплекта. Позиция шаблона — ТИП оборудования:
// при сборке комплекта по шаблону слоты заполняются типами, а конкретные единицы
// выбираются со склада. Состав правки приходит целиком и заменяет прежний.
export const EquipmentTemplateDialog: FC<Props> = ({ open, onClose, template }) => {
	const theme = useTheme()
	// На телефоне диалог шаблона — во весь экран.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const isEdit = Boolean(template?.id)

	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const types = typesData?.data ?? []

	const [create, { isLoading: isCreating }] = useCreateEquipmentTemplateMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentTemplateMutation()

	const { control, handleSubmit, formState } = useForm<FormValues>({
		values: template
			? {
				name: template.name,
				description: template.description,
				items: (template.items ?? []).map(item => ({ typeId: item.typeId })),
			}
			: { name: '', description: '', items: [] },
		mode: 'onTouched',
	})

	const { fields, append, remove } = useFieldArray({ control, name: 'items' })

	const isLoading = isCreating || isUpdating

	const saveHandler = handleSubmit(async form => {
		const items = form.items.filter(item => Boolean(item.typeId))
		if (items.length === 0) {
			toast.error('Добавьте хотя бы один тип оборудования', { autoClose: false })
			return
		}

		const dto: IEquipmentKitTemplateDTO = {
			id: template?.id,
			name: form.name,
			description: form.description,
			items: items.map((item, index) => ({ typeId: item.typeId, sortOrder: index })),
		}

		try {
			if (dto.id) {
				await update(dto).unwrap()
				toast.success('Шаблон обновлён')
			} else {
				await create(dto).unwrap()
				toast.success('Шаблон создан')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при сохранении', { autoClose: false })
		}
	})

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='sm'
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '12px' } } } }}
		>
			<DialogTitle sx={{ m: 0, px: { xs: 1.5, sm: 3 }, py: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' sx={{ fontWeight: 700 }}>
					{isEdit ? 'Редактирование шаблона' : 'Новый шаблон комплекта'}
				</Typography>
				<IconButton onClick={onClose} size='small'>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 2.5 } }}>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						<Field label='Название' required>
							<Controller
								control={control}
								name='name'
								rules={{ required: 'Обязательное поле' }}
								render={({ field, fieldState }) => (
									<TextField
										{...field}
										fullWidth
										size='small'
										placeholder='Например: Стандартное рабочее место'
										error={Boolean(fieldState.error)}
										helperText={fieldState.error?.message}
										sx={{ '& .MuiOutlinedInput-root': { borderRadius: '8px' } }}
									/>
								)}
							/>
						</Field>

						<Field label='Описание'>
							<Controller
								control={control}
								name='description'
								render={({ field }) => (
									<TextField
										{...field}
										fullWidth
										size='small'
										multiline
										minRows={2}
										placeholder='Ноутбук, монитор, мышь, клавиатура'
										sx={{ '& .MuiOutlinedInput-root': { borderRadius: '8px' } }}
									/>
								)}
							/>
						</Field>

						<Box>
							<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}>
								Состав шаблона *
							</Typography>

							<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
								{fields.map((item, index) => (
									<Box key={item.id} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
										<Controller
											control={control}
											name={`items.${index}.typeId`}
											render={({ field }) => (
												<TextField
													{...field}
													select
													fullWidth
													size='small'
													sx={{ '& .MuiOutlinedInput-root': { borderRadius: '8px' } }}
												>
													{types.map(type => (
														<MenuItem key={type.id} value={type.id}>
															{type.name}
														</MenuItem>
													))}
												</TextField>
											)}
										/>
										<IconButton size='small' onClick={() => remove(index)}>
											<Trash2Icon sx={{ fontSize: 16, color: '#9ca3af' }} />
										</IconButton>
									</Box>
								))}
							</Box>

							<Button
								type='button'
								onClick={() => append({ typeId: types[0]?.id ?? '' })}
								startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
								sx={{ mt: 1, textTransform: 'none' }}
							>
								Добавить тип
							</Button>
							<FormHelperText>Из этих типов собирается состав комплекта</FormHelperText>
						</Box>
					</Box>
				</DialogContent>

				<DialogActions sx={{ px: { xs: 1.5, sm: 3 }, py: 2, gap: 1, justifyContent: 'flex-end' }}>
					<Button
						type='button'
						onClick={onClose}
						variant='outlined'
						sx={{ textTransform: 'none', color: '#374151', borderColor: '#d1d5db', borderRadius: '8px' }}
					>
						Отмена
					</Button>
					<Button
						type='submit'
						variant='contained'
						disabled={isLoading || !formState.isValid || (isEdit && !formState.isDirty)}
						sx={{ textTransform: 'none', px: 3, borderRadius: '8px', bgcolor: '#0432a5', '&:hover': { bgcolor: '#002d91' } }}
					>
						{isEdit ? 'Сохранить' : 'Создать'}
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}
