import { type FC } from 'react'
import {
	Dialog,
	DialogContent,
	DialogTitle,
	Box,
	TextField,
	Button,
	Typography,
	Divider,
	DialogActions,
	Autocomplete,
	IconButton,
} from '@mui/material'
import { FormProvider, useForm, Controller, useWatch } from 'react-hook-form'
import { RotateCwIcon, TriangleAlertIcon, InfoIcon, XIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { IFetchError } from '@/app/types/error'
import type { IEquipmentTypePeriodicTaskDTO, IEquipmentType } from '../../types/equipment'
import type { Priority } from '@/features/tasks/types/task'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { pluralize } from '@/utils/plural'
import {
	useGetPeriodicTaskTemplateQuery,
	useCreatePeriodicTaskTemplateMutation,
	useUpdatePeriodicTaskTemplateMutation,
	useGetAllEquipmentTypesQuery,
} from '../../equipmentApiSlice'
import { getRealm } from '@/features/realms/realmSlice'
import { CategoryField } from '@/components/FormFields'
import { Switch } from '@/components/Switch/Switch'

type Props = {
	open: boolean
	onClose: () => void
	templateId: string | null
}

const EMPTY_FORM: IEquipmentTypePeriodicTaskDTO = {
	realmId: '',
	equipmentTypeId: '',
	periodYears: 0,
	periodMonths: 0,
	periodDays: 0,
	title: '',
	description: '',
	priority: 'medium',
	categoryId: '',
	siteId: undefined,
	autoStartOnFirstAssign: false,
	enabled: true,
}

// Живая сводка периода
const PeriodSummary: FC = () => {
	const [years = 0, months = 0, days = 0] = useWatch({
		name: ['periodYears', 'periodMonths', 'periodDays'],
	}).map(v => Number(v) || 0)

	const parts = [
		years > 0 && `${years} ${pluralize(years, ['год', 'года', 'лет'])}`,
		months > 0 && `${months} ${pluralize(months, ['месяц', 'месяца', 'месяцев'])}`,
		days > 0 && `${days} ${pluralize(days, ['день', 'дня', 'дней'])}`,
	].filter(Boolean)

	const isEmpty = parts.length === 0

	const styles = isEmpty
		? { bgcolor: '#fffbeb', borderColor: '#fde68a', color: '#92400e' }
		: { bgcolor: '#eff6ff', borderColor: '#bfdbfe', color: '#1e40af' }

	const Icon = isEmpty ? TriangleAlertIcon : RotateCwIcon

	return (
		<Box
			sx={{
				mt: 2.5,
				display: 'flex',
				alignItems: 'center',
				gap: 2,
				borderRadius: '8px',
				px: 2,
				py: 1.5,
				border: '1px solid',
				...styles,
			}}
		>
			<Icon sx={{ fontSize: 16, flexShrink: 0 }} />
			<Typography component={'p'} variant='caption' sx={{ color: 'inherit', lineHeight: 1 }}>
				{isEmpty ? (
					'Период не задан — укажите хотя бы одно значение'
				) : (
					<>
						Заявка будет создаваться{' '}
						<Box component='span' sx={{ fontWeight: 600 }}>
							каждые {parts.join(', ')}
						</Box>{' '}
						после предыдущего запуска
					</>
				)}
			</Typography>
		</Box>
	)
}

export const PeriodicTaskDialog: FC<Props> = ({ open, onClose, templateId }) => {
	const realm = useAppSelector(getRealm)
	const isEditing = templateId !== null && templateId !== ''

	const { data: existingTemplate } = useGetPeriodicTaskTemplateQuery(templateId ?? '', { skip: !templateId })
	const { data: typesData } = useGetAllEquipmentTypesQuery()
	const equipmentTypes = typesData?.data ?? []

	const [createTemplate] = useCreatePeriodicTaskTemplateMutation()
	const [updateTemplate] = useUpdatePeriodicTaskTemplateMutation()

	const methods = useForm<IEquipmentTypePeriodicTaskDTO>({
		mode: 'onChange',
		defaultValues: EMPTY_FORM,
		values: isEditing ? existingTemplate?.data : { ...EMPTY_FORM, realmId: realm?.id ?? '' },
	})
	const { handleSubmit, formState } = methods
	const { isValid } = formState

	const onSubmit = async (data: IEquipmentTypePeriodicTaskDTO) => {
		const payload: IEquipmentTypePeriodicTaskDTO = {
			...data,
			periodYears: Number(data.periodYears) || 0,
			periodMonths: Number(data.periodMonths) || 0,
			periodDays: Number(data.periodDays) || 0,
		}

		try {
			if (isEditing && data.id) {
				await updateTemplate(payload).unwrap()
				toast.success('Шаблон обновлён')
			} else {
				await createTemplate(payload).unwrap()
				toast.success('Шаблон создан')
			}
			onClose()
		} catch (error) {
			const fetchError = error as IFetchError
			toast.error(fetchError.data?.message || 'Ошибка при сохранении', { autoClose: false })
		}
	}

	return (
		<Dialog
			open={open}
			onClose={onClose}
			maxWidth='md'
			fullWidth
			slotProps={{
				paper: { sx: { borderRadius: { xs: 0, sm: '12px' } } },
			}}
		>
			<DialogTitle
				sx={{
					m: 0,
					px: { xs: 1.5, sm: 3 },
					py: 2,
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
				}}
			>
				<Box>
					<Typography variant='h6' sx={{ fontWeight: 700 }}>
						{isEditing ? 'Редактирование шаблона' : 'Новый шаблон периодической задачи'}
					</Typography>
					<Typography variant='caption' sx={{ color: '#6b7280', display: 'block', fontSize: '12px' }}>
						{isEditing
							? 'Измените параметры шаблона'
							: 'Будет применяться ко всему оборудованию выбранного типа'}
					</Typography>
				</Box>
				<IconButton onClick={onClose}>
					<XIcon sx={{ fontSize: 18 }} />
				</IconButton>
			</DialogTitle>

			<FormProvider {...methods}>
				<Box
					component='form'
					onSubmit={handleSubmit(onSubmit)}
					sx={{ display: 'flex', flexDirection: 'column', flex: 1 }}
				>
					<DialogContent dividers sx={{ px: { xs: 1.5, sm: 3 }, py: { xs: 2, sm: 2.5 } }}>
						{/* Основные параметры */}
						<Typography
							variant='caption'
							sx={{
								display: 'block',
								fontWeight: 600,
								textTransform: 'uppercase',
								letterSpacing: '0.05em',
								color: '#6b7280',
								mb: 1.5,
								fontSize: '12px',
							}}
						>
							Основные параметры
						</Typography>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Название *
								</Typography>
								<Controller
									name='title'
									rules={{ required: 'Название обязательно' }}
									render={({ field, fieldState }) => (
										<TextField
											{...field}
											fullWidth
											size='small'
											error={!!fieldState.error}
											helperText={fieldState.error?.message}
											placeholder='Например: Плановое ТО ноутбуков'
										/>
									)}
								/>
							</Box>
							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Описание
								</Typography>
								<Controller
									name='description'
									render={({ field }) => (
										<TextField
											{...field}
											fullWidth
											multiline
											rows={3}
											placeholder='Что нужно сделать при каждом запуске: чистка, диагностика, замена расходников...'
										/>
									)}
								/>
							</Box>
							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Тип оборудования *
								</Typography>
								<Controller
									name='equipmentTypeId'
									rules={{ required: 'Выберите тип оборудования' }}
									render={({ field, fieldState }) => (
										<Autocomplete
											{...field}
											fullWidth
											options={equipmentTypes}
											getOptionLabel={(type: IEquipmentType) => type.name}
											value={equipmentTypes.find(t => t.id === field.value) || null}
											onChange={(_e, val) => field.onChange(val?.id || '')}
											renderInput={params => (
												<TextField
													{...params}
													error={Boolean(fieldState.error)}
													helperText={fieldState.error?.message}
													placeholder='— Выберите тип —'
												/>
											)}
										/>
									)}
								/>
								<Typography
									variant='caption'
									sx={{ color: '#6b7280', mt: 1, display: 'block', fontSize: '12px' }}
								>
									Задача будет создана для каждой единицы этого типа
								</Typography>
							</Box>
						</Box>

						<Divider sx={{ borderColor: '#f3f4f6', my: 2 }} />

						{/* Периодичность */}
						<Typography
							variant='caption'
							sx={{
								display: 'block',
								fontWeight: 600,
								textTransform: 'uppercase',
								letterSpacing: '0.05em',
								color: '#6b7280',
								mb: 1.5,
								fontSize: '12px',
							}}
						>
							Периодичность
						</Typography>
						<Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 1 }}>
							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Лет
								</Typography>
								<Controller
									name='periodYears'
									render={({ field }) => <TextField {...field} type='number' fullWidth />}
								/>
							</Box>

							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Месяцев
								</Typography>
								<Controller
									name='periodMonths'
									render={({ field }) => <TextField {...field} type='number' fullWidth />}
								/>
							</Box>

							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Дней
								</Typography>
								<Controller
									name='periodDays'
									render={({ field }) => <TextField {...field} type='number' fullWidth />}
								/>
							</Box>
						</Box>
						<PeriodSummary />

						<Divider sx={{ borderColor: '#f3f4f6', my: 2 }} />

						{/* Параметры заявки */}
						<Typography
							variant='caption'
							sx={{
								display: 'block',
								fontWeight: 600,
								textTransform: 'uppercase',
								letterSpacing: '0.05em',
								color: '#6b7280',
								mb: 1.5,
								fontSize: '12px',
							}}
						>
							Параметры создаваемой заявки
						</Typography>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Категория *
								</Typography>
								<CategoryField name='categoryId' rules={{ required: 'Обязательное поле' }} />
							</Box>

							<Box>
								<Typography
									variant='caption'
									sx={{ fontWeight: 600, mb: 0.5, display: 'block', color: '#374151' }}
								>
									Приоритет
								</Typography>
								<Controller
									name='priority'
									render={({ field }) => (
										<Autocomplete
											{...field}
											fullWidth
											options={Object.keys(PRIORITY_MAP) as Priority[]}
											getOptionLabel={(value: Priority) => PRIORITY_MAP[value].label}
											value={field.value || null}
											onChange={(_e, val) => field.onChange(val || 'medium')}
											renderInput={params => <TextField {...params} />}
										/>
									)}
								/>
							</Box>
						</Box>

						<Divider sx={{ borderColor: '#f3f4f6', my: 2 }} />

						{/* Настройки */}
						<Typography
							variant='caption'
							sx={{
								display: 'block',
								fontWeight: 600,
								textTransform: 'uppercase',
								letterSpacing: '0.05em',
								color: '#6b7280',
								mb: 1.5,
								fontSize: '12px',
							}}
						>
							Настройки
						</Typography>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
							<Box
								component='label'
								sx={{
									display: 'flex',
									alignItems: 'center',
									gap: 2,
									py: 1,
									px: 2,
									border: '1px solid #e5e7eb',
									borderRadius: '8px',
									cursor: 'pointer',
									'&:hover': { bgcolor: '#f9fafb' },
									'&:has(input:checked)': {
										borderColor: '#60a5fa',
										bgcolor: '#eff6ff',
									},
								}}
							>
								<Controller
									name='autoStartOnFirstAssign'
									render={({ field }) => <Switch {...field} checked={field.value} />}
								/>
								<Box sx={{ flex: 1 }}>
									<Typography
										variant='subtitle2'
										sx={{ fontWeight: 500, color: '#111827', fontSize: '14px' }}
									>
										Автостарт при первой выдаче
									</Typography>
									<Typography
										sx={{
											color: '#6b7280',
											display: 'block',
											fontSize: '12px',
										}}
									>
										Отсчёт начнётся от даты выдачи: первый запуск = выдача + период. Если выключено
										или даты выдачи нет — первый запуск только вручную.
									</Typography>
								</Box>
							</Box>

							<Box
								component='label'
								sx={{
									display: 'flex',
									alignItems: 'center',
									gap: 2,
									py: 1,
									px: 2,
									border: '1px solid #e5e7eb',
									borderRadius: '8px',
									cursor: 'pointer',
									'&:hover': { bgcolor: '#f9fafb' },
									'&:has(input:checked)': {
										borderColor: '#60a5fa',
										bgcolor: '#eff6ff',
									},
								}}
							>
								<Controller
									name='enabled'
									render={({ field }) => <Switch {...field} checked={field.value} />}
								/>
								<Box sx={{ flex: 1 }}>
									<Typography
										variant='subtitle2'
										sx={{ fontWeight: 500, color: '#111827', fontSize: '14px' }}
									>
										Шаблон активен
									</Typography>
									<Typography
										sx={{
											color: '#6b7280',
											display: 'block',
											fontSize: '12px',
										}}
									>
										Отключённый шаблон не создаёт новые задачи
									</Typography>
								</Box>
							</Box>
						</Box>
					</DialogContent>

					<DialogActions
						sx={{
							px: { xs: 1.5, sm: 3 },
							py: 2,
							gap: 1,
							justifyContent: 'space-between',
						}}
					>
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
							<InfoIcon sx={{ fontSize: 14, color: '#9ca3af' }} />
							<Typography variant='caption' sx={{ color: '#6b7280', fontSize: '12px' }}>
								{isValid ? 'Все обязательные поля заполнены' : 'Заполните обязательные поля'}
							</Typography>
						</Box>
						<Box sx={{ display: 'flex', gap: 1 }}>
							<Button
								onClick={onClose}
								variant='outlined'
								sx={{
									textTransform: 'none',
									color: '#374151',
									borderColor: '#d1d5db',
									borderRadius: '8px',
								}}
							>
								Отмена
							</Button>
							<Button
								type='submit'
								variant='contained'
								disabled={!isValid}
								sx={{
									textTransform: 'none',
									px: 3,
									borderRadius: '8px',
									bgcolor: '#0432a5',
									'&:hover': { bgcolor: '#002d91' },
								}}
							>
								{isEditing ? 'Сохранить' : 'Создать'}
							</Button>
						</Box>
					</DialogActions>
				</Box>
			</FormProvider>
		</Dialog>
	)
}
