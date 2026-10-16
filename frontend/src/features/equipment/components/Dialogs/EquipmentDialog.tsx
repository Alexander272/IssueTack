import { useState, type FC } from 'react'
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
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, FormProvider, useForm } from 'react-hook-form'
import { Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import {
	useCreateEquipmentMutation,
	useDeleteEquipmentMutation,
	useUpdateEquipmentMutation,
} from '../../equipmentApiSlice'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'
import { IdentityFields } from './UnitForm/IdentityFields'
import { OwnershipFields } from './UnitForm/OwnershipFields'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

type Props = {
	open: boolean
	onClose: () => void
	unit?: IEquipment | null
	// isKitDefault — при создании (без unit) сразу отметить «Комплект».
	isKitDefault?: boolean
	// onDeleted — единицу удалили: вызывающему мало закрыть форму, карточку
	// единицы тоже нужно убрать (иначе она осталась бы пустой).
	onDeleted?: () => void
}

// Значения формы: при редактировании — из единицы, при создании — пустые.
// typeId у комплекта равен null, поэтому в форме это пустая строка.
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
	notes: unit?.notes ?? '',
	isKit: unit?.isKit ?? isKitDefault,
})

// Диалог создания/редактирования единицы оборудования: держит форму, её
// сохранение и удаление единицы (в режиме правки, через подтверждение), а поля
// раскладывает по секциям UnitForm. Секции берут справочники и права из формы и
// своих запросов — контейнеру их протаскивать не нужно.
export const EquipmentDialog: FC<Props> = ({ open, onClose, unit, isKitDefault, onDeleted }) => {
	const theme = useTheme()
	// На телефоне форма на всю ширину — как TaskCreateModal и карточка DetailModal.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const isEdit = Boolean(unit?.id)

	const canDelete = useCheckPermissions({ anyOf: [PermRules.Equipment.Delete] })
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [create, { isLoading: isCreating }] = useCreateEquipmentMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()

	const methods = useForm<IEquipmentDTO>({
		defaultValues: toDefault(unit, isKitDefault),
		mode: 'onTouched',
	})
	const { control, handleSubmit, formState } = methods

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async (form: IEquipmentDTO) => {
		const payload: IEquipmentDTO = {
			...form,
			// У комплекта типа нет: не шлём пустую строку, иначе бэкенд не разберёт
			// её как UUID. У единицы тип обязателен (поле «Тип» в форме).
			typeId: form.isKit ? undefined : form.typeId || undefined,
			serialNumber: form.serialNumber || null,
			inventoryNumber: form.inventoryNumber || null,
			holderName: form.holderName || null,
			location: form.location || null,
			purchaseDate: form.purchaseDate || null,
			warrantyUntil: form.warrantyUntil || null,
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

	const handleDelete = async () => {
		if (!unit?.id) return
		try {
			await remove(unit.id).unwrap()
			toast.success('Оборудование удалено')
			setDeleteOpen(false)
			onClose()
			onDeleted?.()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при удалении', { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='sm'
			slotProps={{
				paper: { sx: { borderRadius: { xs: 0, sm: '16px' }, p: { xs: 0, sm: 1 } } },
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
									render={({ field }) => (
										<TextField {...field} multiline minRows={2} {...inputProps} />
									)}
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
						{/* Удаление живёт здесь, рядом с сохранением: `mr: 'auto'`
						    отжимает кнопку влево, а `type='button'` не даёт ей
						    отправить форму. */}
						{isEdit && canDelete && (
							<Button
								type='button'
								variant='outlined'
								color='error'
								onClick={() => setDeleteOpen(true)}
								disabled={isDeleting}
								startIcon={<Trash2Icon sx={{ fontSize: 18 }} />}
								sx={{ textTransform: 'none', mr: 'auto' }}
							>
								Удалить
							</Button>
						)}
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

			{/* Подтверждение удаления — отдельным диалогом поверх формы, чтобы
			    промах по кнопке не удалял единицу сразу. */}
			<ConfirmDialog
				open={deleteOpen}
				title='Удаление оборудования'
				message={`Вы уверены, что хотите удалить «${unit?.name ?? ''}»? Действие необратимо.`}
				confirmLabel='Удалить'
				confirmColor='error'
				onConfirm={handleDelete}
				onCancel={() => setDeleteOpen(false)}
				loading={isDeleting}
			/>
		</Dialog>
	)
}
