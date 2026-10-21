import { useState, type FC, type ReactNode } from 'react'
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
	MenuItem,
	Stack,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import dayjs from 'dayjs'
import { CheckIcon, ChevronDownIcon, Trash2Icon, WarehouseIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { DateTextField } from '@/components/DatePicker/DatePicker'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import {
	useCreateEquipmentMutation,
	useDeleteEquipmentMutation,
	useGetAllEquipmentShopsQuery,
	useGetAllEquipmentTypesQuery,
	useUpdateEquipmentMutation,
} from '../../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'
import { SpecsEditor } from './UnitForm/SpecsEditor'
import { pairsToSpecs, specsToPairs, type ISpecPair } from './UnitForm/specs'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

type Props = {
	open: boolean
	onClose: () => void
	unit?: IEquipment | null
	// onDeleted — единицу удалили: вызывающему мало закрыть форму, карточку
	// единицы тоже нужно убрать (иначе она осталась бы пустой).
	onDeleted?: () => void
}

// Значения формы: при редактировании — из единицы, при создании — пустые.
// Скрытые поля (vendor, model, parentId, holderUserId, holderName, isKit)
// по-прежнему заполняются из единицы и уходят в payload — правка не должна
// молча стирать их, даже если соответствующих инпутов в форме нет.
// Характеристики (specs) правятся списком пар, поэтому в форме живут массивом
// (см. SpecsEditor); в объект их собирает saveHandler при отправке.
const toDefault = (unit?: IEquipment | null): IEquipmentDTO => ({
	id: unit?.id,
	typeId: unit?.typeId ?? '',
	name: unit?.name ?? '',
	vendor: unit?.vendor ?? '',
	model: unit?.model ?? '',
	serialNumber: unit?.serialNumber ?? '',
	inventoryNumber: unit?.inventoryNumber ?? '',
	status: unit?.status ?? 'in_stock',
	shop: unit?.shop ?? '',
	orderNumber: unit?.orderNumber ?? '',
	parentId: unit?.parentId ?? null,
	siteId: unit?.siteId ?? null,
	holderUserId: unit?.holderUserId ?? null,
	holderName: unit?.holderName ?? '',
	location: unit?.location ?? '',
	purchaseDate: unit?.purchaseDate ?? '',
	warrantyUntil: unit?.warrantyUntil ?? '',
	notes: unit?.notes ?? '',
	isKit: unit?.isKit ?? false,
})

// Заголовок секции формы: мелкая ЗАГЛАВНАЯ подпись-разделитель (как в макете).
const SectionLabel: FC<{ children: ReactNode }> = ({ children }) => (
	<Typography
		variant='caption'
		sx={{
			display: 'block',
			color: 'text.secondary',
			fontWeight: 600,
			textTransform: 'uppercase',
			letterSpacing: '0.04em',
			mb: 1.5,
		}}
	>
		{children}
	</Typography>
)

// Диалог создания/редактирования единицы оборудования. Раскладка — по макету:
// обязательные поля в первой секции, затем «Расположение» и «Дополнительно»,
// между секциями — разделители. «Комплект», «Часть комплекта», «Производитель»
// и «Модель» из формы убраны: комплекты ведутся в своей модалке, а имя/модель
// живут в наименовании. Их значения по-прежнему сохраняются молча.
// В «Дополнительно» — реквизиты закупки (магазин, номер заказа) и свободные
// характеристики (SpecsEditor, список пар).
// Комплекты сюда не попадают: создание и правка комплекта — EquipmentKitDialog
// (одна и та же форма в обоих режимах), DetailHeader выбирает диалог по isKit.
export const EquipmentDialog: FC<Props> = ({ open, onClose, unit, onDeleted }) => {
	const theme = useTheme()
	// На телефоне форма на всю ширину — как TaskCreateModal и карточка DetailModal.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const isEdit = Boolean(unit?.id)

	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const { data: sitesData } = useGetAllSitesQuery()
	const { data: shopsData } = useGetAllEquipmentShopsQuery()
	const types = typesData?.data ?? []
	const sites = sitesData?.data ?? []
	// Подсказки для поля «Магазин»: справочник реалма, а не уникальные значения
	// из единиц — список переживает удаление единицы и не «загрязняется».
	const shops = (shopsData?.data ?? []).map(s => s.name)

	const canDelete = useCheckPermissions({ anyOf: [PermRules.Equipment.Delete] })
	const [deleteOpen, setDeleteOpen] = useState(false)
	const [create, { isLoading: isCreating }] = useCreateEquipmentMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentMutation()
	const [remove, { isLoading: isDeleting }] = useDeleteEquipmentMutation()

	const methods = useForm<IEquipmentDTO>({
		defaultValues: toDefault(unit),
		mode: 'onTouched',
	})
	const { control, handleSubmit, formState } = methods

	// Характеристики держим вне RHF: список пар — динамическая структура, и
	// объект specs не пережил бы ввод пустого ключа (две строки с одним ключом).
	const [specsPairs, setSpecsPairs] = useState<ISpecPair[]>(() => specsToPairs(unit?.specs))

	const isLoading = isCreating || isUpdating || isDeleting

	const saveHandler = handleSubmit(async (form: IEquipmentDTO) => {
		const payload: IEquipmentDTO = {
			...form,
			// specs собираем из пар: пустые строки отбрасываются, ключи подрезаны.
			specs: pairsToSpecs(specsPairs),
			// Не шлём пустую строку вместо UUID: тип обязателен у единицы,
			// а форма комплект сюда больше не попадает.
			typeId: form.typeId || undefined,
			serialNumber: form.serialNumber || null,
			inventoryNumber: form.inventoryNumber || null,
			holderName: form.holderName || null,
			shop: form.shop || null,
			orderNumber: form.orderNumber || null,
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

	// Две колонки на десктопе, одна на телефоне — как в макете.
	const grid = { display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 2 }

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='md'
			slotProps={{
				paper: {
					sx: {
						borderRadius: { xs: 0, sm: '16px' },
						// Максимальная высота как в макете (max-h-[90vh]) на
						// десктопе: контент прокручивается, шапка и футер
						// остаются на месте. На телефоне окно открывается во
						// весь экран (fullScreen), и MUI сам ставит maxHeight:
						// none — наш sx побеждал бы его, и окно открывалось бы
						// на 90% высоты с полями сверху и снизу.
						maxHeight: { xs: 'none', sm: '90vh' },
					},
				},
			}}
		>
			<DialogTitle
				// На телефоне окно во весь экран: отступы уменьшены до 12px по
				// горизонтали и 16px по вертикали вместо 16/20. С sm — как было.
				sx={{
					m: 0,
					px: { xs: 1.5, sm: 2 },
					py: { xs: 2, sm: 2 },
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'flex-start',
				}}
			>
				<Stack>
					<Typography variant='h6' sx={{ fontWeight: 'bold' }}>
						{isEdit ? 'Редактирование оборудования' : 'Новое оборудование'}
					</Typography>
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						{isEdit ? 'Обновите данные единицы' : 'Оборудование будет добавлено на склад'}
					</Typography>
				</Stack>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 3 } }}>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
						{/* Основная информация: тип, название, номера */}
						<Box sx={grid}>
							<Field label='Тип оборудования' required>
								<Controller
									control={control}
									name='typeId'
									rules={{ required: 'Выберите тип' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											select
											{...inputProps}
											// Заглушка: у MUI Select пустое значение без displayEmpty
											// не подставляет пункт с value='', а renderValue для
											// пустого не вызывается вовсе — нужны оба.
											slotProps={{
												select: {
													displayEmpty: true,
													renderValue: value =>
														value === '' || value == null
															? '— Выберите тип —'
															: (types.find(t => t.id === value)?.name ?? ''),
												},
											}}
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										>
											<MenuItem value='' disabled>
												— Выберите тип —
											</MenuItem>
											{types.map(t => (
												<MenuItem key={t.id} value={t.id}>
													{t.name}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Field>

							<Field label='Название' required>
								<Controller
									control={control}
									name='name'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											{...inputProps}
											placeholder='Например: Dell OptiPlex 7090'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										/>
									)}
								/>
							</Field>

							<Field label='Серийный номер'>
								<Controller
									control={control}
									name='serialNumber'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='CN-0XYZ-12345' />
									)}
								/>
							</Field>

							<Field label='Инвентарный номер'>
								<Controller
									control={control}
									name='inventoryNumber'
									render={({ field }) => (
										<TextField
											{...field}
											{...inputProps}
											placeholder='Оставьте пустым для автогенерации'
										/>
									)}
								/>
							</Field>
						</Box>

						<Divider />

						{/* Расположение: площадка (обязательна при создании) и локация */}
						<Box>
							<SectionLabel>Расположение</SectionLabel>
							<Box sx={grid}>
								{/* Площадка обязательна при создании: без держателя и локации
								    CHECK «держатель ИЛИ место» не пропустит пустую единицу. На
								    правке оставляем необязательным — у места может быть держатель. */}
								<Field label='Площадка' required={!isEdit}>
									<Controller
										control={control}
										name='siteId'
										rules={isEdit ? undefined : { required: 'Укажите площадку' }}
										render={({ field, fieldState }) => (
											<TextField
												{...field}
												select
												{...inputProps}
												value={field.value ?? ''}
												// Пустое значение не показывает пункт-заглушку само
												// (renderValue для пустого не вызывается без
												// displayEmpty) — нужны оба.
												slotProps={{
													select: {
														displayEmpty: true,
														renderValue: value =>
															value === '' || value == null
																? '— Выберите площадку —'
																: (sites.find(s => s.id === value)?.name ?? ''),
													},
												}}
												error={Boolean(fieldState.error)}
												helperText={fieldState.error?.message}
											>
												<MenuItem value=''>— Выберите площадку —</MenuItem>
												{sites.map(s => (
													<MenuItem key={s.id} value={s.id}>
														{s.name}
													</MenuItem>
												))}
											</TextField>
										)}
									/>
								</Field>

								<Field label='Расположение'>
									<Controller
										control={control}
										name='location'
										render={({ field }) => (
											<TextField {...field} {...inputProps} placeholder='Склад, стеллаж А-3' />
										)}
									/>
								</Field>
							</Box>
						</Box>

						<Divider />

						{/* Дополнительно: закупка, даты, характеристики и примечание */}
						<Box>
							<SectionLabel>Дополнительно</SectionLabel>
							<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
								<Box sx={grid}>
									<Field label='Магазин'>
										<Controller
											control={control}
											name='shop'
											render={({ field }) => (
												/* freeSolo: магазин выбирают из справочника или
												   вписывают новое название руками. Новое значение
												   уйдёт в справочник тем же сохранением
												   (EnsureShop на бэкенде в транзакции единицы). */
												<Autocomplete
													freeSolo
													options={shops}
													value={field.value || null}
													onChange={(_e, value) => field.onChange(value ?? '')}
													// Стрелка-маркер списка: у freeSolo можно и вписать
													// руками, поэтому индикатор не MUI-шный по умолчанию,
													// а явный — иначе поле выглядит как обычный ввод.
													popupIcon={
														<ChevronDownIcon
															sx={{ fontSize: 18, color: 'text.secondary' }}
														/>
													}
													renderInput={params => (
														<TextField
															{...params}
															{...inputProps}
															placeholder='Выберите или укажите магазин'
														/>
													)}
												/>
											)}
										/>
									</Field>

									<Field label='Номер заказа'>
										<Controller
											control={control}
											name='orderNumber'
											render={({ field }) => (
												<TextField
													{...field}
													{...inputProps}
													placeholder='Номер заказа/накладной'
												/>
											)}
										/>
									</Field>
								</Box>

								<Box sx={grid}>
									<Field label='Дата покупки'>
										<Controller
											control={control}
											name='purchaseDate'
											render={({ field }) => (
												<DatePicker
													value={field.value ? dayjs(field.value) : null}
													onChange={date =>
														field.onChange(date ? date.format('YYYY-MM-DD') : null)
													}
													slots={{ textField: DateTextField }}
													slotProps={{ textField: { fullWidth: true, size: 'small' } }}
												/>
											)}
										/>
									</Field>

									<Field label='Гарантия до'>
										<Controller
											control={control}
											name='warrantyUntil'
											render={({ field }) => (
												<DatePicker
													value={field.value ? dayjs(field.value) : null}
													onChange={date =>
														field.onChange(date ? date.format('YYYY-MM-DD') : null)
													}
													slots={{ textField: DateTextField }}
													slotProps={{ textField: { fullWidth: true, size: 'small' } }}
												/>
											)}
										/>
									</Field>
								</Box>

								{/* Статус в форме не выбирается: его выводят операции (выдача →
								    «в эксплуатации», возврат → «на складе»), а остальные
								    значения (дефект, списание) — будущий отдельный механизм.
								    Поле «Статус» убрано вместе с селектом. Сохранённое значение
								    всё равно уходит в payload (из toDefault), поэтому правка
								    статус не сбрасывает. */}

								<Field label='Характеристики'>
									<SpecsEditor pairs={specsPairs} onChange={setSpecsPairs} />
								</Field>

								<Field label='Примечание'>
									<Controller
										control={control}
										name='notes'
										render={({ field }) => (
											<TextField
												{...field}
												{...inputProps}
												multiline
												minRows={2}
												placeholder='Состояние, комплектация, особенности...'
											/>
										)}
									/>
								</Field>
							</Box>
						</Box>
					</Box>
				</DialogContent>

				{/* Футер серый, слева — подсказка о складе (или удаление на правке),
				    справа — отмена/сохранение. */}
				<DialogActions
					sx={{
						justifyContent: 'space-between',
						bgcolor: 'grey.50',
						px: { xs: 1.5, sm: 2 },
						py: 2,
					}}
				>
					{/* Слева — подсказка о складе при создании или удаление при правке;
					    без прав на удаление слот пустой, чтобы кнопки остались справа. */}
					{isEdit ? (
						canDelete ? (
							<Button
								type='button'
								variant='outlined'
								color='error'
								onClick={() => setDeleteOpen(true)}
								disabled={isDeleting}
								startIcon={<Trash2Icon sx={{ fontSize: 18 }} />}
								sx={{ textTransform: 'none' }}
							>
								Удалить
							</Button>
						) : (
							<Box />
						)
					) : (
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, color: 'text.secondary' }}>
							<WarehouseIcon sx={{ fontSize: 16, color: 'primary.main' }} />
							<Typography variant='body2'>Будет добавлено на склад</Typography>
						</Box>
					)}

					<Box sx={{ display: 'flex', gap: 1 }}>
						<Button
							type='button'
							onClick={onClose}
							variant='outlined'
							sx={{ textTransform: 'none', color: 'text.primary', borderColor: 'grey.300' }}
						>
							Отмена
						</Button>
						<Button
							type='submit'
							variant='contained'
							disabled={isLoading || !formState.isValid}
							startIcon={isEdit ? undefined : <CheckIcon sx={{ fontSize: 16 }} />}
							sx={{ textTransform: 'none', px: 3 }}
						>
							{isEdit ? 'Сохранить' : 'Создать'}
						</Button>
					</Box>
				</DialogActions>
			</form>

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
