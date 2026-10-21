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
	IconButton,
	MenuItem,
	Stack,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useForm, useWatch } from 'react-hook-form'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import dayjs from 'dayjs'
import { InfoIcon, UserPlusIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { DateTextField } from '@/components/DatePicker/DatePicker'
import type { IAssignEquipmentDTO, IEquipment } from '../../types/equipment'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'
import { useGetRealmUsersQuery } from '@/features/user/usersApiSlice'
import { useAssignEquipmentMutation } from '../../equipmentApiSlice'
import { resolveTypeIcon } from '../../typeIcons'
import { EquipmentStatusBadge } from '../EquipmentStatusBadge'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

// Значения формы: место — реквизит диалога, а не DTO. Момент выдачи хранится
// в assignedAt строкой ISO (её даёт DateTimePicker), skipAct выводится из
// чекбокса «Создать акт выдачи» (по умолчанию акт формируется).
type AssignForm = {
	userId: string | null
	holderName: string
	// recipientName — снимок ФИО получателя для акта (без username): ФИО
	// выбранного сотрудника либо свободный текст. Отдельным полем, потому что
	// holderName пуст при выборе из справочника, а акту нужна подпись.
	recipientName: string
	siteId: string | null
	location: string
	comment: string
	createAct: boolean
	assignedAt: string | null
	// returnDueAt — предполагаемый срок возврата. Пусто = выдают бессрочно.
	returnDueAt: string | null
}

const userLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full ? `${full} (${u.username})` : u.username
}

// recipientLabel — ФИО получателя для акта: без username, как в снимке имени на
// сервере (last_name + first_name, fallback — username). Так подпись в акте
// совпадает с той, что выводится в остальных местах.
const recipientLabel = (u: { lastName: string; firstName: string; username: string }) => {
	const full = `${u.lastName} ${u.firstName}`.trim()
	return full || u.username
}

// Вариант freeSolo — либо сотрудник из справочника, либо набранная строка.
const optionLabel = (option: string | { lastName: string; firstName: string; username: string }) =>
	typeof option === 'string' ? option : userLabel(option)

// Диалог выдачи оборудования — по макету: шапка с подписью, карточка того, что
// выдаём (иконка типа, статус, тип · инв. № · S/N), получатель, площадка и
// расположение, дата выдачи, комментарий и опции (акт, принудительное
// переназначение). Связь с заявкой и уведомления получателю пока не заведены.
export const EquipmentAssignDialog: FC<Props> = ({ open, onClose, unit }) => {
	const theme = useTheme()
	// На телефоне диалог выдачи — во весь экран (как форма создания заявки).
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const [assign, { isLoading }] = useAssignEquipmentMutation()
	const { data: sitesData } = useGetAllSitesQuery()
	// Получателем может быть любой пользователь реалма, не только участник
	// групп: оборудование выдают и заказчикам, и сотрудникам без заявок,
	// поэтому справочник запрашивается без фильтра членства.
	const { data: usersData } = useGetRealmUsersQuery('all')

	const sites = sitesData?.data ?? []
	const users = usersData?.data ?? []

	// Место выдаётся тем же, что у единицы: выдача — про держателя, а не про
	// переезд. Оператор может переписать оба поля, тогда место сменится.
	const methods = useForm<AssignForm>({
		defaultValues: {
			userId: null,
			holderName: '',
			recipientName: '',
			siteId: unit.siteId ?? null,
			location: unit.location ?? '',
			comment: '',
			createAct: true,
			assignedAt: dayjs().toISOString(),
			returnDueAt: null,
		},
		mode: 'onTouched',
	})
	const { control, handleSubmit } = methods

	const createAct = useWatch({ control, name: 'createAct' })

	// Подпись карточки: у комплекта нет типа и серийного номера, поэтому пустые
	// части отбрасываются, а не оставляют « · » на конце.
	const subtitle = [
		unit.isKit ? '' : unit.typeName,
		unit.inventoryNumber && `Инв. №${unit.inventoryNumber}`,
		unit.serialNumber && `S/N: ${unit.serialNumber}`,
	]
		.filter(Boolean)
		.join(' · ')

	const icon = resolveTypeIcon({ isKit: unit.isKit, icon: unit.typeIcon, typeName: unit.typeName })

	const saveHandler = handleSubmit(async form => {
		const payload: IAssignEquipmentDTO = {
			userId: form.userId || null,
			holderName: form.holderName || null,
			recipientName: form.recipientName || form.holderName || null,
			siteId: form.siteId || null,
			location: form.location || null,
			comment: form.comment,
			skipAct: !form.createAct,
			assignedAt: form.assignedAt ?? undefined,
			returnDueAt: form.returnDueAt,
		}
		try {
			await assign({ id: unit.id, body: payload }).unwrap()
			toast.success('Оборудование выдано')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при выдаче', { autoClose: false })
		}
	})

	return (
		<Dialog
			open={open}
			onClose={onClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='sm'
			slotProps={{
				paper: {
					sx: {
						borderRadius: { xs: 0, sm: '16px' },
						// Как в макете (max-h-[90vh]) на десктопе: прокручивается
						// контент, шапка и футер остаются на месте. На телефоне
						// окно во весь экран (fullScreen), и MUI сам ставит
						// maxHeight: none — наш sx побеждал бы его, и окно
						// открывалось бы на 90% высоты с полями.
						maxHeight: { xs: 'none', sm: '90vh' },
					},
				},
			}}
		>
			<DialogTitle
				// На телефоне окно во весь экран: 12px по горизонтали и 16px по
				// вертикали вместо 16/20. С sm — как было.
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
						Выдача оборудования
					</Typography>
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						Передача оборудования сотруднику под подпись
					</Typography>
				</Stack>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 2 } }}>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						{/* Что выдаём: иконка типа в тоне реестра, название со
						    статусом и мелкая подпись с номерами. */}
						<Box
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1.5,
								p: 2,
								borderRadius: '10px',
								border: '1px solid',
								borderColor: 'primary.main',
								bgcolor: 'rgba(234, 238, 252, 0.5)',
							}}
						>
							<Box
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
									width: 48,
									height: 48,
									borderRadius: '8px',
									bgcolor: icon.bg,
									flexShrink: 0,
								}}
							>
								<icon.Icon sx={{ fontSize: 22, color: icon.color }} />
							</Box>
							<Box sx={{ minWidth: 0, flex: 1 }}>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
									<Typography sx={{ fontSize: '0.875rem', fontWeight: 600 }}>{unit.name}</Typography>
									<EquipmentStatusBadge status={unit.status} />
								</Box>
								{subtitle && (
									<Typography variant='caption' sx={{ color: 'text.secondary' }}>
										{subtitle}
									</Typography>
								)}
							</Box>
						</Box>

						{/* Получатель: сотрудник из справочника либо свободное ФИО —
						    выдать можно и внешнему сотруднику без учётки. Одно поле
						    с freeSolo: выбранный вариант приходит объектом IUserData
						    (уходит в userId), набранный руками текст — строкой
						    (уходит в holderName). */}
						<Field label='Получатель' required>
							<Controller
								control={control}
								name='userId'
								rules={{
									validate: value =>
										Boolean(value || methods.getValues('holderName')) ||
										'Выберите сотрудника или укажите его ФИО',
								}}
								render={({ field, fieldState }) => (
									<Autocomplete
										freeSolo
										options={users}
										getOptionLabel={optionLabel}
										// Сотрудник опознаётся по ссылке на объект из
										// options; иначе в поле показывается введённое
										// ФИО ('' вместо null — свободное значение).
										value={users.find(u => u.id === field.value) ?? (field.value ? '' : null)}
										onChange={(_e, value) => {
											if (value && typeof value === 'object') {
												field.onChange(value.id)
												methods.setValue('holderName', '')
												// Акт берёт подпись получателя отсюда: при
												// выборе из справочника holderName пуст.
												methods.setValue('recipientName', recipientLabel(value))
												return
											}
											const text = typeof value === 'string' ? value : ''
											field.onChange(null)
											methods.setValue('holderName', text)
											methods.setValue('recipientName', text)
										}}
										renderInput={params => (
											<TextField
												{...params}
												{...inputProps}
												placeholder='— Выберите сотрудника или укажите ФИО —'
												error={Boolean(fieldState.error)}
												helperText={fieldState.error?.message}
											/>
										)}
									/>
								)}
							/>
						</Field>

						{/* Площадка и расположение заполнены текущим местом единицы: пустые
						    поля в выдаче означают «единица не переезжает». */}
						<Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
							<Field label='Площадка' required>
								<Controller
									control={control}
									name='siteId'
									rules={{ required: 'Укажите площадку' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											select
											{...inputProps}
											value={field.value ?? ''}
											// Заглушка: renderValue для пустого значения не
											// вызывается без displayEmpty — нужны оба.
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
										<TextField {...field} {...inputProps} placeholder='Каб. 301' />
									)}
								/>
							</Field>
						</Stack>

						<Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
							<Field label='Дата и время выдачи' required>
								<Controller
									control={control}
									name='assignedAt'
									rules={{ required: 'Укажите дату и время' }}
									render={({ field, fieldState }) => (
										<DateTimePicker
											value={field.value ? dayjs(field.value) : null}
											onChange={date => field.onChange(date ? date.toISOString() : null)}
											slots={{ textField: DateTextField }}
											slotProps={{
												textField: {
													fullWidth: true,
													size: 'small',
													error: Boolean(fieldState.error),
													helperText: fieldState.error?.message,
												},
											}}
										/>
									)}
								/>
							</Field>

							{/* Предполагаемый срок возврата: необязателен (без него выдают
						    бессрочно). В этот день выдавшему придёт напоминание в
						    Mattermost. Проверка «не раньше выдачи» дублирует серверную
						    (EQ019) — чтобы ошибку было видно до отправки. */}
							<Field label='Предполагаемый срок возврата'>
								<Controller
									control={control}
									name='returnDueAt'
									rules={{
										validate: value => {
											if (!value) return true
											const assignedAt = methods.getValues('assignedAt')
											if (assignedAt && dayjs(value).isBefore(dayjs(assignedAt), 'day')) {
												return 'Срок возврата не может быть раньше даты выдачи'
											}
											return true
										},
									}}
									render={({ field, fieldState }) => (
										<DatePicker
											value={field.value ? dayjs(field.value) : null}
											onChange={date => field.onChange(date ? date.toISOString() : null)}
											slots={{ textField: DateTextField }}
											slotProps={{
												textField: {
													fullWidth: true,
													size: 'small',
													error: Boolean(fieldState.error),
													// Подсказка до ошибки: срок необязателен, и без
													// пояснения пустое поле читается как «забыли».
													helperText:
														fieldState.error?.message ?? 'Напомним о возврате в этот день',
												},
												field: { clearable: true },
											}}
										/>
									)}
								/>
							</Field>
						</Stack>

						<Field label='Комментарий'>
							<Controller
								control={control}
								name='comment'
								render={({ field }) => (
									<TextField
										{...field}
										{...inputProps}
										multiline
										minRows={2}
										placeholder='Причина выдачи, особенности...'
									/>
								)}
							/>
						</Field>

						{/* Опции — как в макете: строка с чекбоксом и пояснением.
						    Уведомление получателю пока не реализовано. */}
						<Box sx={{ borderTop: '1px solid', borderColor: 'divider', pt: 2.5 }}>
							<Box
								sx={{ display: 'flex', alignItems: 'flex-start', gap: 1, cursor: 'pointer' }}
								onClick={() => methods.setValue('createAct', !createAct, { shouldDirty: true })}
							>
								<Controller
									control={control}
									name='createAct'
									render={({ field }) => (
										<Checkbox
											checked={Boolean(field.value)}
											onChange={e => field.onChange(e.target.checked)}
											sx={{ mt: -0.75, ml: -0.75 }}
										/>
									)}
								/>
								<Box>
									<Typography sx={{ fontSize: '0.875rem', fontWeight: 500 }}>
										Создать акт выдачи
									</Typography>
									<Typography variant='caption' sx={{ color: 'text.secondary' }}>
										Автоматически сформировать акт для подписания
									</Typography>
								</Box>
							</Box>
						</Box>
					</Box>
				</DialogContent>

				{/* Футер серый: слева — что произойдёт со статусом, справа — кнопки. */}
				<DialogActions
					sx={{ justifyContent: 'space-between', bgcolor: 'grey.50', px: { xs: 1.5, sm: 2 }, py: 2 }}
				>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: 'text.secondary' }}>
						<InfoIcon sx={{ fontSize: 16, color: 'info.main' }} />
						<Typography variant='body2'>Статус изменится на «В эксплуатации»</Typography>
					</Box>

					<Box sx={{ display: 'flex', gap: 1 }}>
						<Button
							type='button'
							onClick={onClose}
							variant='outlined'
							sx={{
								textTransform: 'none',
								color: 'text.primary',
								borderColor: 'grey.300',
								backgroundColor: 'white',
							}}
						>
							Отмена
						</Button>
						<Button
							type='submit'
							variant='contained'
							disabled={isLoading}
							sx={{ textTransform: 'none', px: 3 }}
						>
							<UserPlusIcon sx={{ fontSize: 18, mr: 1 }} />
							Выдать
						</Button>
					</Box>
				</DialogActions>
			</form>
		</Dialog>
	)
}
