import { type FC } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Checkbox,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControlLabel,
	IconButton,
	MenuItem,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment, IEquipmentDTO } from '../types/equipment'
import { EQUIPMENT_STATUS_OPTIONS } from '../types/equipment'
import {
	useCreateEquipmentMutation,
	useGetAllEquipmentQuery,
	useGetAllEquipmentTypesQuery,
	useUpdateEquipmentMutation,
} from '../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	unit?: IEquipment | null
}

const fieldLabelSx = { fontWeight: 600, mb: 0.5, display: 'block' }
const inputProps = { size: 'small' as const, fullWidth: true }

const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

const toDefault = (unit?: IEquipment | null): IEquipmentDTO => ({
	id: unit?.id,
	typeId: unit?.typeId ?? '',
	name: unit?.name ?? '',
	vendor: unit?.vendor ?? '',
	model: unit?.model ?? '',
	serialNumber: unit?.serialNumber ?? '',
	inventoryNumber: unit?.inventoryNumber ?? '',
	status: unit?.status ?? 'in_stock',
	parentId: unit?.parentId ?? null,
	siteId: unit?.siteId ?? null,
	holderUserId: unit?.holderUserId ?? null,
	holderName: unit?.holderName ?? '',
	location: unit?.location ?? '',
	purchaseDate: unit?.purchaseDate ?? '',
	warrantyUntil: unit?.warrantyUntil ?? '',
	cost: unit?.cost ?? null,
	notes: unit?.notes ?? '',
	isKit: unit?.isKit ?? false,
})

export const EquipmentDialog: FC<Props> = ({ open, onClose, unit }) => {
	const isEdit = Boolean(unit?.id)

	const [create, { isLoading: isCreating }] = useCreateEquipmentMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentMutation()

	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const { data: sitesData } = useGetAllSitesQuery()
	const { data: usersData } = useGetRealmUsersQuery('executors')
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })

	const types = typesData?.data ?? []
	const sites = sitesData?.data ?? []
	const users = usersData?.data ?? []
	const parentCandidates = (unitsData?.data ?? []).filter(u => u.id !== unit?.id)

	const { control, handleSubmit, formState } = useForm<IEquipmentDTO>({
		defaultValues: toDefault(unit),
		mode: 'onTouched',
	})

	const isLoading = isCreating || isUpdating

	const saveHandler = handleSubmit(async (form: IEquipmentDTO) => {
		const payload: IEquipmentDTO = {
			...form,
			serialNumber: form.serialNumber || null,
			inventoryNumber: form.inventoryNumber || null,
			holderName: form.holderName || null,
			location: form.location || null,
			purchaseDate: form.purchaseDate || null,
			warrantyUntil: form.warrantyUntil || null,
			cost: Number.isFinite(Number(form.cost)) ? Number(form.cost) : null,
			parentId: form.parentId || null,
			siteId: form.siteId || null,
			holderUserId: form.holderUserId || null,
		}
		try {
			if (isEdit) {
				await update(payload).unwrap()
				toast.success('Оборудование обновлено')
			} else {
				await create(payload).unwrap()
				toast.success('Оборудование создано')
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
			maxWidth='sm'
			slotProps={{
				paper: { sx: { borderRadius: '16px', p: 1 } },
			}}
		>
			<DialogTitle
				sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
			>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					{isEdit ? 'Редактирование оборудования' : 'Новое оборудование'}
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent
					dividers
					sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
				>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Название{' '}
									<Typography component='span' color='error'>
										*
									</Typography>
								</Typography>
								<Controller
									control={control}
									name='name'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											{...inputProps}
											placeholder='Например: Системный блок'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Тип{' '}
									<Typography component='span' color='error'>
										*
									</Typography>
								</Typography>
								<Controller
									control={control}
									name='typeId'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											select
											{...inputProps}
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										>
											<MenuItem value='' disabled>
												Выберите тип
											</MenuItem>
											{types.map(t => (
												<MenuItem key={t.id} value={t.id}>
													{t.name}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Статус
								</Typography>
								<Controller
									control={control}
									name='status'
									render={({ field }) => (
										<TextField {...field} select {...inputProps}>
											{EQUIPMENT_STATUS_OPTIONS.map(s => (
												<MenuItem key={s.value} value={s.value}>
													{s.label}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Производитель
								</Typography>
								<Controller
									control={control}
									name='vendor'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='Например: Lenovo' />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Модель
								</Typography>
								<Controller
									control={control}
									name='model'
									render={({ field }) => <TextField {...field} {...inputProps} />}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Серийный номер
								</Typography>
								<Controller
									control={control}
									name='serialNumber'
									render={({ field }) => <TextField {...field} {...inputProps} />}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Инвентарный номер
								</Typography>
								<Controller
									control={control}
									name='inventoryNumber'
									render={({ field }) => <TextField {...field} {...inputProps} />}
								/>
							</Box>
						</Box>

						<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }}>
							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Площадка
								</Typography>
								<Controller
									control={control}
									name='siteId'
									render={({ field }) => (
										<TextField {...field} select {...inputProps} value={field.value ?? ''}>
											<MenuItem value=''>Не указана</MenuItem>
											{sites.map(s => (
												<MenuItem key={s.id} value={s.id}>
													{s.name}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Локация на площадке
								</Typography>
								<Controller
									control={control}
									name='location'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='Склад, полка A' />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Держатель (сотрудник)
								</Typography>
								<Controller
									control={control}
									name='holderUserId'
									render={({ field }) => (
										<Autocomplete
											options={users}
											getOptionLabel={userLabel}
											value={users.find(u => u.id === field.value) ?? null}
											onChange={(_e, value) => field.onChange(value?.id ?? null)}
											renderInput={params => (
												<TextField
													{...params}
													size='small'
													placeholder='Выберите сотрудника'
												/>
											)}
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Держатель (название)
								</Typography>
								<Controller
									control={control}
									name='holderName'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='Или свободный текст' />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Дата покупки
								</Typography>
								<Controller
									control={control}
									name='purchaseDate'
									render={({ field }) => (
										<TextField {...field} type='date' {...inputProps} slotProps={{ inputLabel: { shrink: true } }} />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Гарантия до
								</Typography>
								<Controller
									control={control}
									name='warrantyUntil'
									render={({ field }) => (
										<TextField {...field} type='date' {...inputProps} slotProps={{ inputLabel: { shrink: true } }} />
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Стоимость, ₽
								</Typography>
								<Controller
									control={control}
									name='cost'
									render={({ field }) => (
										<TextField
											{...field}
											type='number'
											{...inputProps}
											onChange={e => field.onChange(e.target.value === '' ? null : e.target.value)}
										/>
									)}
								/>
							</Box>

							<Box>
								<Typography variant='caption' sx={fieldLabelSx}>
									Часть комплекта
								</Typography>
								<Controller
									control={control}
									name='parentId'
									render={({ field }) => (
										<Autocomplete
											options={parentCandidates}
											getOptionLabel={u => u.name}
											value={parentCandidates.find(u => u.id === field.value) ?? null}
											onChange={(_e, value) => field.onChange(value?.id ?? null)}
											renderInput={params => (
												<TextField
													{...params}
													size='small'
													placeholder='Родитель (необязательно)'
												/>
											)}
										/>
									)}
								/>
							</Box>
						</Box>

						<Box>
							<Typography variant='caption' sx={fieldLabelSx}>
								Примечание
							</Typography>
							<Controller
								control={control}
								name='notes'
								render={({ field }) => (
									<TextField {...field} multiline minRows={2} {...inputProps} />
								)}
							/>
						</Box>

						<Controller
							control={control}
							name='isKit'
							render={({ field }) => (
								<FormControlLabel
									control={
										<Checkbox
											checked={Boolean(field.value)}
											onChange={e => field.onChange(e.target.checked)}
										/>
									}
									label='Это комплект (можно вести логический состав и выдавать вместе)'
								/>
							)}
						/>
					</Box>
				</DialogContent>

				<DialogActions sx={{ p: 2, gap: 1, justifyContent: 'flex-end' }}>
					<Button
						type='button'
						onClick={onClose}
						variant='outlined'
						sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
					>
						Отмена
					</Button>
					<Button
						type='submit'
						variant='contained'
						disabled={isLoading || !formState.isValid}
						sx={{ textTransform: 'none', px: 3 }}
					>
						{isEdit ? 'Сохранить' : 'Создать'}
					</Button>
				</DialogActions>
			</form>
		</Dialog>
	)
}