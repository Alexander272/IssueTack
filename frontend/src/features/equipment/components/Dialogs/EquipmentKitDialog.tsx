import { useState, type FC } from 'react'
import {
	Autocomplete,
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	MenuItem,
	Stack,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form'
import { LayersIcon, PackageOpenIcon, PlusIcon, SparklesIcon, Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import {
	useCreateEquipmentMutation,
	useDeleteEquipmentMutation,
	useGetAllEquipmentQuery,
	useGetAllEquipmentTemplatesQuery,
	useGetAllEquipmentTypesQuery,
	useUpdateEquipmentMutation,
} from '../../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'

// Слот состава: тип (из шаблона или выбранный вручную) и конкретная единица со
// склада. Позиция шаблона сразу «замораживает» тип — единицу подбирают под него;
// ручной слот тип меняет свободно.
type KitRow = {
	typeId: string
	unitId: string
	locked: boolean
}

type FormValues = {
	name: string
	inventoryNumber: string
	siteId: string
	location: string
	notes: string
	templateId: string
	rows: KitRow[]
}

type Props = {
	open: boolean
	onClose: () => void
	// unit — правка существующего комплекта (карточка, DetailHeader). Без unit —
	// создание: тогда показывается сборка состава из шаблонов и слотов.
	unit?: IEquipment | null
	// onDeleted — комплект удалён: карточку надо убрать (как в EquipmentDialog).
	onDeleted?: () => void
}

// Модалка комплекта: создание и правка реквизитов. Создание шлёт комплект и
// стартовый состав одним запросом (POST /equipment принимает components), чтобы
// при сбое не остался комплект без состава. Правка те же поля prefill'ит и шлёт
// в PUT — состав в правке не показывается, он редактируется отдельным диалогом
// «Состав комплекта» (PUT components игнорирует). У комплекта нет типа, а
// площадка обязательна, чтобы не упереться в CHECK «держатель ИЛИ место».
export const EquipmentKitDialog: FC<Props> = ({ open, onClose, unit, onDeleted }) => {
	const theme = useTheme()
	// На телефоне модалка комплекта — во весь экран.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const isEdit = Boolean(unit?.id)

	const canDelete = useCheckPermissions({ anyOf: [PermRules.Equipment.Delete] })
	const [deleteOpen, setDeleteOpen] = useState(false)
	const { data: sitesData } = useGetAllSitesQuery()
	// Состав и шаблоны нужны только при создании: в правке их секции нет.
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 }, { skip: isEdit })
	const { data: templatesData } = useGetAllEquipmentTemplatesQuery(undefined, { skip: isEdit })
	const { data: typesData } = useGetAllEquipmentTypesQuery(undefined, { skip: isEdit })
	const [create, { isLoading: isCreating }] = useCreateEquipmentMutation()
	const [update, { isLoading: isUpdating }] = useUpdateEquipmentMutation()
	// removeKit, а не remove: useFieldArray ниже уже держит свой remove.
	const [removeKit, { isLoading: isDeleting }] = useDeleteEquipmentMutation()

	const sites = sitesData?.data ?? []
	// Кандидаты в состав — единицы на складе, комплект в комплект не кладём.
	const stockUnits = (unitsData?.data ?? []).filter(u => !u.isKit && u.status === 'in_stock')
	const templates = templatesData?.data ?? []
	const types = typesData?.data ?? []

	const { control, handleSubmit, formState, setValue } = useForm<FormValues>({
		defaultValues: {
			name: unit?.name ?? '',
			inventoryNumber: unit?.inventoryNumber ?? '',
			siteId: unit?.siteId ?? '',
			location: unit?.location ?? '',
			notes: unit?.notes ?? '',
			templateId: '',
			rows: [],
		},
		mode: 'onTouched',
	})

	const isLoading = isCreating || isUpdating || isDeleting

	const { fields, append, remove, replace } = useFieldArray({ control, name: 'rows' })
	const rows = useWatch({ control, name: 'rows' }) ?? []
	const templateId = useWatch({ control, name: 'templateId' })

	// Уже выбранные единицы не предлагаем дважды: одна единица — один слот.
	const takenUnitIds = new Set(rows.map(row => row.unitId).filter(Boolean))

	const unitOptions = (row: KitRow): IEquipment[] =>
		stockUnits.filter(u => u.typeId === row.typeId && (!takenUnitIds.has(u.id) || u.id === row.unitId))

	const applyTemplate = (templateId: string) => {
		setValue('templateId', templateId)
		if (!templateId) {
			replace([])
			return
		}
		const template = templates.find(t => t.id === templateId)
		if (!template) return
		replace(template.items.map(item => ({ typeId: item.typeId, unitId: '', locked: true })))
	}

	const addBlankRow = () => append({ typeId: types[0]?.id ?? '', unitId: '', locked: false })

	const saveHandler = handleSubmit(async form => {
		if (isEdit && unit) {
			// Правка: UPDATE на бэкенде пишет все колонки разом, поэтому
			// скрытые из формы поля (vendor/model/серийник/даты/держатель/specs/
			// статус) уходят в payload сохранёнными значениями — форма не должна
			// их молча стирать. Состав (components) в PUT игнорируется.
			const payload: IEquipmentDTO = {
				id: unit.id,
				name: form.name,
				inventoryNumber: form.inventoryNumber || null,
				siteId: form.siteId || null,
				location: form.location || null,
				notes: form.notes,
				isKit: true,
				vendor: unit.vendor,
				model: unit.model,
				serialNumber: unit.serialNumber,
				specs: unit.specs,
				status: unit.status,
				purchaseDate: unit.purchaseDate,
				warrantyUntil: unit.warrantyUntil,
				holderUserId: unit.holderUserId,
				holderName: unit.holderName,
				parentId: unit.parentId,
			}
			try {
				await update(payload).unwrap()
				toast.success('Комплект обновлён')
				onClose()
			} catch (error) {
				const fetchError = error as IFetchError
				toast.error(fetchError.data?.message || 'Ошибка при обновлении', { autoClose: false })
			}
			return
		}

		const payload: IEquipmentDTO = {
			name: form.name,
			inventoryNumber: form.inventoryNumber || null,
			siteId: form.siteId || null,
			location: form.location || null,
			notes: form.notes,
			status: 'in_stock',
			isKit: true,
			components: form.rows.filter(row => row.unitId).map(row => ({ componentId: row.unitId })),
		}

		try {
			await create(payload).unwrap()
			toast.success('Комплект создан')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при создании', { autoClose: false })
		}
	})

	const handleDelete = async () => {
		if (!unit?.id) return
		try {
			await removeKit(unit.id).unwrap()
			toast.success('Комплект удалён')
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
			maxWidth='md'
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '16px' } } } }}
		>
			<DialogTitle
				sx={{
					m: 0,
					px: { xs: 1.5, sm: 2 },
					py: 2,
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
				}}
			>
				<Stack>
					<Typography variant='h6' sx={{ fontWeight: 'bold' }}>
						{isEdit ? 'Редактирование комплекта' : 'Новый комплект'}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						{isEdit
							? 'Реквизиты комплекта; состав правится в карточке'
							: 'Комплект и его состав сохраняются вместе'}
					</Typography>
				</Stack>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 2 } }}>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						{/* Основные сведения о комплекте */}
						<Box
							sx={{
								display: 'grid',
								gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
								gap: 2,
							}}
						>
							<Field label='Название комплекта' required>
								<Controller
									control={control}
									name='name'
									rules={{ required: 'Обязательное поле' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											{...inputProps}
											placeholder='Например: Рабочее место №12'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										/>
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

							<Field label='Площадка' required>
								<Controller
									control={control}
									name='siteId'
									rules={{ required: 'Укажите площадку' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											{...inputProps}
											select
											placeholder='Площадка'
											error={Boolean(fieldState.error)}
											helperText={fieldState.error?.message}
										>
											{sites.map(site => (
												<MenuItem key={site.id} value={site.id}>
													{site.name}
												</MenuItem>
											))}
										</TextField>
									)}
								/>
							</Field>

							<Field label='Локация'>
								<Controller
									control={control}
									name='location'
									render={({ field }) => (
										<TextField {...field} {...inputProps} placeholder='Аудитория, кабинет' />
									)}
								/>
							</Field>

							<Box sx={{ gridColumn: { xs: 'span 1', sm: 'span 2' } }}>
								<Field label='Описание'>
									<Controller
										control={control}
										name='notes'
										render={({ field }) => (
											<TextField
												{...field}
												{...inputProps}
												multiline
												minRows={2}
												placeholder='Краткое описание назначения комплекта...'
											/>
										)}
									/>
								</Field>
							</Box>
						</Box>

						{/* Сборка из шаблона: блок скрыт, пока шаблонов нет вовсе. */}
						{templates.length > 0 && (
							<Box
								sx={{
									border: '1px solid #e0e7ff',
									bgcolor: '#f5f7ff',
									borderRadius: '12px',
									p: 2,
									display: 'flex',
									flexDirection: 'column',
									gap: 1,
								}}
							>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
									<SparklesIcon sx={{ fontSize: 18, color: '#4338ca' }} />
									<Typography variant='subtitle2' sx={{ fontWeight: 600, color: '#312e81' }}>
										Собрать из шаблона
									</Typography>
								</Box>
								<Typography variant='caption' sx={{ color: '#4b5563' }}>
									Шаблон заполнит состав типами оборудования — единицы можно подобрать сразу или позже
								</Typography>
								<TextField
									select
									size='small'
									fullWidth
									value={templateId}
									onChange={e => applyTemplate(e.target.value)}
									sx={{ bgcolor: '#fff', borderRadius: '8px', mt: 0.5 }}
								>
									<MenuItem value=''>Без шаблона</MenuItem>
									{templates.map(template => (
										<MenuItem key={template.id} value={template.id}>
											{template.name}
											{template.items.length > 0 ? ` · ${template.items.length}` : ''}
										</MenuItem>
									))}
								</TextField>
							</Box>
						)}

						{/* Состав комплекта — только при создании: в правке он редактируется
						    отдельным диалогом «Состав комплекта» в карточке. */}
						{!isEdit && (
							<Box>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
									<LayersIcon sx={{ fontSize: 18, color: '#374151' }} />
									<Typography variant='subtitle2' sx={{ fontWeight: 600, color: '#374151' }}>
										Состав комплекта
									</Typography>
								</Box>

								{fields.length === 0 ? (
									<Box
										sx={{
											border: '1px dashed #d1d5db',
											borderRadius: '12px',
											px: 2,
											py: 3,
											textAlign: 'center',
											color: '#6b7280',
										}}
									>
										<Typography variant='body2'>Состав пока пуст</Typography>
										<Typography variant='caption'>
											Добавьте компоненты или примените шаблон выше
										</Typography>
									</Box>
								) : (
									<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
										{fields.map((item, index) => {
											const row = rows[index] ?? { typeId: '', unitId: '', locked: false }
											const selectedUnit = stockUnits.find(u => u.id === row.unitId) ?? null

											return (
												<Box
													key={item.id}
													sx={{
														display: 'grid',
														gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr auto' },
														gap: 1.5,
														alignItems: 'flex-start',
														p: 1.5,
														border: '1px solid #e5e7eb',
														borderRadius: '10px',
														bgcolor: '#fafafa',
													}}
												>
													{/* Тип: из шаблона зафиксирован, вручную — меняется. */}
													<Controller
														control={control}
														name={`rows.${index}.typeId`}
														rules={{ required: true }}
														render={({ field }) => (
															<TextField
																{...field}
																{...inputProps}
																select
																disabled={row.locked}
																placeholder='Тип оборудования'
																helperText={row.locked ? 'Из шаблона' : undefined}
																onChange={e => {
																	field.onChange(e)
																	// Смена типа обнуляет единицу: прежняя к нему не подходит.
																	setValue(`rows.${index}.unitId`, '')
																}}
															>
																{types.map(type => (
																	<MenuItem key={type.id} value={type.id}>
																		{type.name}
																	</MenuItem>
																))}
															</TextField>
														)}
													/>

													<Controller
														control={control}
														name={`rows.${index}.unitId`}
														render={({ field }) => (
															<Autocomplete
																options={unitOptions(row)}
																getOptionLabel={u => u.name}
																isOptionEqualToValue={(option, value) =>
																	option.id === value.id
																}
																value={selectedUnit}
																disabled={!row.typeId}
																onChange={(_e, value) => field.onChange(value?.id ?? '')}
																renderInput={params => (
																	<TextField
																		{...params}
																		size='small'
																		placeholder='Конкретная единица (со склада)'
																	/>
																)}
															/>
														)}
													/>

													<IconButton onClick={() => remove(index)} sx={{ mt: 0.25 }}>
														<Trash2Icon sx={{ fontSize: 18, color: '#9ca3af' }} />
													</IconButton>
												</Box>
											)
										})}
									</Box>
								)}

								<Button
									type='button'
									onClick={addBlankRow}
									startIcon={<PlusIcon sx={{ fontSize: 16 }} />}
									sx={{ mt: 1.5, textTransform: 'none' }}
								>
									Добавить компонент вручную
								</Button>

								<Typography variant='caption' sx={{ display: 'block', color: '#9ca3af' }}>
									Пустой слот можно заполнить позже в карточке комплекта
								</Typography>
							</Box>
						)}
					</Box>
				</DialogContent>

				{/* Футер: слева — счётчик состава при создании или удаление при
				    правке (слот пустой без прав, чтобы кнопки остались справа). */}
				<DialogActions sx={{ px: { xs: 1.5, sm: 2 }, py: 2, gap: 1, justifyContent: 'space-between' }}>
					{isEdit ? (
						canDelete ? (
							<Button
								type='button'
								variant='outlined'
								color='error'
								onClick={() => setDeleteOpen(true)}
								disabled={isDeleting}
								startIcon={<Trash2Icon sx={{ fontSize: 16 }} />}
								sx={{ textTransform: 'none' }}
							>
								Удалить
							</Button>
						) : (
							<Box />
						)
					) : (
						<Typography variant='body2' sx={{ color: '#6b7280' }}>
							Позиций в составе: {fields.length}
						</Typography>
					)}
					<Box sx={{ display: 'flex', gap: 1 }}>
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
							startIcon={isEdit ? undefined : <PackageOpenIcon sx={{ fontSize: 16 }} />}
							sx={{ textTransform: 'none', px: 3 }}
						>
							{isEdit ? 'Сохранить' : 'Создать комплект'}
						</Button>
					</Box>
				</DialogActions>
			</form>

			{/* Подтверждение удаления — отдельным диалогом поверх формы. */}
			<ConfirmDialog
				open={deleteOpen}
				title='Удаление комплекта'
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
