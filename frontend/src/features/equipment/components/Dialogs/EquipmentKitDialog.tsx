import { type FC } from 'react'
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
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useFieldArray, useForm, useWatch } from 'react-hook-form'
import { LayersIcon, PackageOpenIcon, PlusIcon, SparklesIcon, Trash2Icon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipment, IEquipmentDTO } from '../../types/equipment'
import {
	useCreateEquipmentMutation,
	useGetAllEquipmentQuery,
	useGetAllEquipmentTemplatesQuery,
	useGetAllEquipmentTypesQuery,
} from '../../equipmentApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'

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
}

// Модалка «Новый комплект». Комплект и его стартовый состав уходят одним
// запросом: POST /equipment принимает components, поэтому при сбое не остаётся
// комплект без состава. У комплекта нет типа, а площадка обязательна, чтобы не
// упереться в CHECK «держатель ИЛИ место».
export const EquipmentKitDialog: FC<Props> = ({ open, onClose }) => {
	const theme = useTheme()
	// На телефоне модалка комплекта — во весь экран.
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))

	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const { data: sitesData } = useGetAllSitesQuery()
	const { data: unitsData } = useGetAllEquipmentQuery({ limit: 500 })
	const { data: templatesData } = useGetAllEquipmentTemplatesQuery()
	const [create, { isLoading }] = useCreateEquipmentMutation()

	const types = typesData?.data ?? []
	const sites = sitesData?.data ?? []
	// Кандидаты в состав — единицы на складе, комплект в комплект не кладём.
	const stockUnits = (unitsData?.data ?? []).filter(u => !u.isKit && u.status === 'in_stock')
	const templates = templatesData?.data ?? []

	const { control, handleSubmit, formState, setValue } = useForm<FormValues>({
		defaultValues: {
			name: '',
			inventoryNumber: '',
			siteId: '',
			location: '',
			notes: '',
			templateId: '',
			rows: [],
		},
		mode: 'onTouched',
	})

	const { fields, append, remove, replace } = useFieldArray({ control, name: 'rows' })
	const rows = useWatch({ control, name: 'rows' }) ?? []
	const templateId = useWatch({ control, name: 'templateId' })

	// Уже выбранные единицы не предлагаем дважды: одна единица — один слот.
	const takenUnitIds = new Set(rows.map(row => row.unitId).filter(Boolean))

	const unitOptions = (row: KitRow): IEquipment[] =>
		stockUnits.filter(
			u => u.typeId === row.typeId && (!takenUnitIds.has(u.id) || u.id === row.unitId),
		)

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

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='md'
			slotProps={{ paper: { sx: { borderRadius: { xs: 0, sm: '16px' } } } }}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Box>
					<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
						Новый комплект
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280' }}>
						Комплект и его состав сохраняются вместе
					</Typography>
				</Box>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent
					dividers
					sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}
				>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
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
										placeholder='Необязательно'
										helperText='Оставьте пустым — номер присвоится автоматически (KIT-…)'
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

					{/* Состав комплекта */}
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
														isOptionEqualToValue={(option, value) => option.id === value.id}
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
				</Box>
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1, justifyContent: 'space-between' }}>
				<Typography variant='body2' sx={{ color: '#6b7280' }}>
					Позиций в составе: {fields.length}
				</Typography>
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
						startIcon={<PackageOpenIcon sx={{ fontSize: 16 }} />}
						sx={{ textTransform: 'none', px: 3 }}
					>
						Создать комплект
					</Button>
				</Box>
			</DialogActions>
			</form>
		</Dialog>
	)
}
