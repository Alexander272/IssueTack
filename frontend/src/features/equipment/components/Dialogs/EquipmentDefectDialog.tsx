import { type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	Stack,
	TextField,
	Typography,
	useMediaQuery,
	useTheme,
} from '@mui/material'
import { Controller, useForm } from 'react-hook-form'
import { DatePicker } from '@mui/x-date-pickers/DatePicker'
import dayjs from 'dayjs'
import { AlertTriangleIcon, InfoIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import { DateTextField } from '@/components/DatePicker/DatePicker'
import type { IAddDefectDTO, IEquipment } from '../../types/equipment'
import { useAddEquipmentDefectMutation } from '../../equipmentApiSlice'
import { resolveTypeIcon } from '../../typeIcons'
import { EquipmentStatusBadge } from '../EquipmentStatusBadge'
import { Field } from './UnitForm/Field'
import { inputProps } from './UnitForm/consts'

type Props = {
	open: boolean
	onClose: () => void
	unit: IEquipment
}

// Значения формы: описание неисправности и момент обнаружения. Дата хранится
// строкой ISO (её даёт DateTimePicker) — в DTO она уходит как есть.
type DefectForm = {
	description: string
	foundAt: string | null
}

// Диалог фиксации дефекта — по стилю выдачи: шапка с подписью, карточка единицы
// (иконка типа, статус, тип · инв. № · S/N), описание и дата обнаружения. Дефект
// переводит единицу в «Дефектное» — из этого состояния начальник области сможет
// её списать. Повторно дефект не заводят: кнопка доступна только у активной единицы.
export const EquipmentDefectDialog: FC<Props> = ({ open, onClose, unit }) => {
	const theme = useTheme()
	// На телефоне диалог разворачивается во весь экран (как выдача).
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const [addDefect, { isLoading }] = useAddEquipmentDefectMutation()

	const methods = useForm<DefectForm>({
		defaultValues: {
			description: '',
			foundAt: dayjs().toISOString(),
		},
		mode: 'onTouched',
	})
	const { control, handleSubmit } = methods

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
		const payload: IAddDefectDTO = {
			description: form.description.trim(),
			foundAt: form.foundAt ?? undefined,
		}
		try {
			await addDefect({ id: unit.id, body: payload }).unwrap()
			toast.success('Дефект зафиксирован')
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при фиксации дефекта', { autoClose: false })
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
						// Как в выдаче: на десктопе прокручивается контент, шапка и
						// футер остаются на месте; на телефоне окно во весь экран.
						maxHeight: { xs: 'none', sm: '90vh' },
					},
				},
			}}
		>
			<DialogTitle
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
						Добавить дефект
					</Typography>
					<Typography variant='caption' sx={{ color: 'text.secondary' }}>
						Фиксация неисправности оборудования
					</Typography>
				</Stack>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<form onSubmit={saveHandler}>
				<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 2 } }}>
					<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
						{/* Что дефектуем: та же карточка, что в выдаче, но в тоне
						    предупреждения — операция «портит» единицу, а не выдаёт. */}
						<Box
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1.5,
								p: 2,
								borderRadius: '10px',
								border: '1px solid',
								borderColor: 'warning.main',
								bgcolor: 'rgba(255, 243, 224, 0.5)',
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

						<Field label='Описание дефекта' required>
							<Controller
								control={control}
								name='description'
								rules={{ required: 'Опишите неисправность' }}
								render={({ field, fieldState }) => (
									<TextField
										{...field}
										{...inputProps}
										multiline
										minRows={3}
										placeholder='Что именно сломалось, как проявляется...'
										error={Boolean(fieldState.error)}
										helperText={fieldState.error?.message}
									/>
								)}
							/>
						</Field>

						{/* Дата обнаружения: по умолчанию сегодня, но дефект могли найти
						    раньше. Только день — время обнаружения никто не фиксирует.
						    Будущее отклоняется (disableFuture и проверка, дублирующая
						    серверную EQ023), чтобы ошибку было видно до отправки. */}
						<Field label='Когда обнаружен' required>
							<Controller
								control={control}
								name='foundAt'
								rules={{
									required: 'Укажите дату обнаружения',
									validate: value =>
										!value || !dayjs(value).isAfter(dayjs(), 'day')
											? true
											: 'Дата обнаружения не может быть в будущем',
								}}
								render={({ field, fieldState }) => (
									<DatePicker
										value={field.value ? dayjs(field.value) : null}
										onChange={date => field.onChange(date ? date.toISOString() : null)}
										slots={{ textField: DateTextField }}
										disableFuture
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
					</Box>
				</DialogContent>

				{/* Футер серый: слева — что произойдёт со статусом, справа — кнопки.
				    На телефоне раскладывается колонкой: в строку подпись и две кнопки
				    не помещаются, и они сжимались бы до нечитаемых. */}
				<DialogActions
					sx={{
						justifyContent: 'space-between',
						bgcolor: 'grey.50',
						px: { xs: 1.5, sm: 2 },
						py: 2,
						flexDirection: { xs: 'column', sm: 'row' },
						alignItems: { xs: 'stretch', sm: 'center' },
						gap: { xs: 1.5, sm: 0 },
					}}
				>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, color: 'text.secondary' }}>
						<InfoIcon sx={{ fontSize: 16, color: 'warning.main' }} />
						<Typography variant='body2'>Статус изменится на «Дефектное»</Typography>
					</Box>

					<Box sx={{ display: 'flex', gap: 1, width: { xs: '100%', sm: 'auto' } }}>
						<Button
							type='button'
							onClick={onClose}
							variant='outlined'
							sx={{
								textTransform: 'none',
								color: 'text.primary',
								borderColor: 'grey.300',
								backgroundColor: 'white',
								flex: { xs: 1, sm: 'none' },
							}}
						>
							Отмена
						</Button>
						<Button
							type='submit'
							variant='contained'
							color='warning'
							disabled={isLoading}
							sx={{ textTransform: 'none', px: 3, flex: { xs: 1, sm: 'none' } }}
						>
							<AlertTriangleIcon sx={{ fontSize: 18, mr: 1 }} />
							Зафиксировать
						</Button>
					</Box>
				</DialogActions>
			</form>
		</Dialog>
	)
}
