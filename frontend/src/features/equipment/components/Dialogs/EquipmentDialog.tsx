import { type FC } from 'react'
import {
	Box,
	Button,
	Checkbox,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	FormControlLabel,
	IconButton,
	TextField,
	Typography,
} from '@mui/material'
import { Controller, FormProvider, useForm } from 'react-hook-form'
import { XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import { useCreateEquipmentMutation, useUpdateEquipmentMutation } from '../../equipmentApiSlice'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'
import { IdentityFields } from './UnitForm/IdentityFields'
import { OwnershipFields } from './UnitForm/OwnershipFields'

type Props = {
	open: boolean
	onClose: () => void
	unit?: IEquipment | null
	// isKitDefault — при создании (без unit) сразу отметить «Комплект».
	isKitDefault?: boolean
}

// Значения формы: при редактировании — из единицы, при создании — пустые.
const toDefault = (unit?: IEquipment | null, isKitDefault = false): IEquipmentDTO => ({
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
	isKit: unit?.isKit ?? isKitDefault,
})

// Диалог создания/редактирования единицы оборудования: держит форму и её
// сохранение, а поля раскладывает по секциям UnitForm. Секции берут справочники
// и права из формы и своих запросов — контейнеру их протаскивать не нужно.
export const EquipmentDialog: FC<Props> = ({ open, onClose, unit, isKitDefault }) => {
	const isEdit = Boolean(unit?.id)

	const [create, { isLoading: isCreating }] = useCreateEquipmentMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentMutation()

	const methods = useForm<IEquipmentDTO>({
		defaultValues: toDefault(unit, isKitDefault),
		mode: 'onTouched',
	})
	const { control, handleSubmit, formState } = methods

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
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					{isEdit ? 'Редактирование оборудования' : 'Новое оборудование'}
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<FormProvider {...methods}>
				<form onSubmit={saveHandler}>
					<DialogContent
						dividers
						sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
					>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<IdentityFields />

							<OwnershipFields />

							<Field label='Примечание'>
								<Controller
									control={control}
									name='notes'
									render={({ field }) => <TextField {...field} multiline minRows={2} {...inputProps} />}
								/>
							</Field>

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
			</FormProvider>
		</Dialog>
	)
}
