import { useMemo, useState, type FC } from 'react'
import {
	Dialog,
	DialogTitle,
	DialogContent,
	DialogActions,
	Button,
	TextField,
	Box,
	Typography,
	FormControl,
	FormLabel,
	RadioGroup,
	FormControlLabel,
	Radio,
	Autocomplete,
	Checkbox,
} from '@mui/material'

import {
	useGetPeriodicTaskTemplateQuery,
	useCreatePeriodicTaskTemplateMutation,
	useUpdatePeriodicTaskTemplateMutation,
} from '../../equipmentApiSlice'
import type { IEquipmentTypePeriodicTaskDTO } from '../../types/equipment'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import type { Priority } from '@/features/tasks/types/task'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllSitesQuery } from '@/features/sites/sitesApiSlice'

type Props = {
	open: boolean
	onClose: () => void
	// templateId — null/undefined: создание шаблона; иначе редактирование.
	templateId: string | null
	realmId: string
	equipmentTypeId: string
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

export const PeriodicTaskDialog: FC<Props> = ({ open, onClose, templateId, realmId, equipmentTypeId }) => {
	const isEditing = templateId !== null && templateId !== ''
	const queryKey = templateId ?? '__new__'

	const { data: existingTemplate } = useGetPeriodicTaskTemplateQuery(templateId ?? '', { skip: !templateId })
	const { data: categoriesData } = useGetAllCategoriesQuery(void 0)
	const { data: sitesData } = useGetAllSitesQuery(void 0)
	const categories = categoriesData?.data ?? []
	const sites = sitesData?.data ?? []

	const [createTemplate] = useCreatePeriodicTaskTemplateMutation()
	const [updateTemplate] = useUpdatePeriodicTaskTemplateMutation()

	// Базовые значения: пустая форма при создании, данные шаблона при редактировании.
	// Пока шаблон грузится — null, форма рендерится пустой (кнопка заблокирована валидацией).
	const baseForm = useMemo<IEquipmentTypePeriodicTaskDTO | null>(() => {
		if (!isEditing) {
			return { ...EMPTY_FORM, realmId, equipmentTypeId }
		}
		const t = existingTemplate
		if (!t) return null
		return {
			id: t.id ?? '',
			realmId: t.realmId,
			equipmentTypeId: t.equipmentTypeId,
			periodYears: t.periodYears,
			periodMonths: t.periodMonths,
			periodDays: t.periodDays,
			title: t.title,
			description: t.description,
			priority: t.priority,
			categoryId: t.categoryId,
			siteId: t.siteId,
			autoStartOnFirstAssign: t.autoStartOnFirstAssign,
			enabled: t.enabled,
		}
	}, [isEditing, existingTemplate, realmId, equipmentTypeId])

	// Перекрытие правок пользователя: пока пользователь не сохранил изменения,
	// форма редактируется поверх базовых значений; сброс происходит при смене шаблона.
	const [overrides, setOverrides] = useState<{ key: string; data: IEquipmentTypePeriodicTaskDTO } | null>(null)
	const formData = overrides && overrides.key === queryKey ? overrides.data : baseForm
	const setFormData = (data: IEquipmentTypePeriodicTaskDTO) => setOverrides({ key: queryKey, data })

	const handleSubmit = async () => {
		if (!formData) return
		const payload: IEquipmentTypePeriodicTaskDTO = { ...formData, realmId, equipmentTypeId }
		try {
			if (templateId && formData.id) {
				await updateTemplate(payload).unwrap()
			} else {
				await createTemplate(payload).unwrap()
			}
			onClose()
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	const isFormValid =
		formData !== null &&
		formData.title.trim() !== '' &&
		formData.categoryId !== '' &&
		(formData.periodYears > 0 || formData.periodMonths > 0 || formData.periodDays > 0)

	if (!formData) {
		return (
			<Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
				<DialogTitle>{templateId ? 'Редактирование шаблона периодической задачи' : 'Новый шаблон периодической задачи'}</DialogTitle>
			</Dialog>
		)
	}

	return (
		<Dialog open={open} onClose={onClose} fullWidth maxWidth='sm'>
			<DialogTitle>
				{templateId ? 'Редактирование шаблона периодической задачи' : 'Новый шаблон периодической задачи'}
			</DialogTitle>

			<DialogContent>
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
					<TextField
						fullWidth
						label='Название'
						value={formData.title}
						onChange={e => setFormData({ ...formData, title: e.target.value })}
						required
					/>

					<TextField
						fullWidth
						label='Описание'
						value={formData.description}
						onChange={e => setFormData({ ...formData, description: e.target.value })}
						multiline
						rows={3}
					/>

					<Box>
						<Typography variant='subtitle2' gutterBottom>
							Периодичность
						</Typography>
						<Box sx={{ display: 'flex', gap: 2, alignItems: 'center' }}>
							<TextField
								label='Лет'
								type='number'
								size='small'
								value={formData.periodYears}
								onChange={e => setFormData({ ...formData, periodYears: Number(e.target.value) || 0 })}
								slotProps={{ input: { inputProps: { min: 0 } } }}
								sx={{ width: 100 }}
							/>
							<TextField
								label='Месяцев'
								type='number'
								size='small'
								value={formData.periodMonths}
								onChange={e => setFormData({ ...formData, periodMonths: Number(e.target.value) || 0 })}
								slotProps={{ input: { inputProps: { min: 0 } } }}
								sx={{ width: 100 }}
							/>
							<TextField
								label='Дней'
								type='number'
								size='small'
								value={formData.periodDays}
								onChange={e => setFormData({ ...formData, periodDays: Number(e.target.value) || 0 })}
								slotProps={{ input: { inputProps: { min: 0 } } }}
								sx={{ width: 100 }}
							/>
						</Box>
					</Box>

					<FormControl>
						<FormLabel id='priority-label'>Приоритет</FormLabel>
						<RadioGroup
							row
							aria-labelledby='priority-label'
							value={formData.priority}
							onChange={e => setFormData({ ...formData, priority: e.target.value as Priority })}
						>
							{(Object.keys(PRIORITY_MAP) as Priority[]).map(value => (
								<FormControlLabel key={value} value={value} control={<Radio />} label={PRIORITY_MAP[value].label} />
							))}
						</RadioGroup>
					</FormControl>

					<Autocomplete
						fullWidth
						options={categories}
						getOptionLabel={(cat: { name: string }) => cat.name}
						value={categories.find(c => c.id === formData.categoryId) || null}
						onChange={(_e, val) => setFormData({ ...formData, categoryId: val?.id ?? '' })}
						renderInput={params => <TextField {...params} label='Категория' required />}
					/>

					<Autocomplete
						fullWidth
						options={sites}
						getOptionLabel={(site: { name: string }) => site.name}
						value={sites.find(s => s.id === formData.siteId) || null}
						onChange={(_e, val) => setFormData({ ...formData, siteId: val?.id })}
						renderInput={params => <TextField {...params} label='Площадка' />}
					/>

					<Box sx={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
						<FormControlLabel
							control={
								<Checkbox checked={formData.autoStartOnFirstAssign} onChange={e => setFormData({ ...formData, autoStartOnFirstAssign: e.target.checked })} />
							}
							label='Автостарт при первой выдаче'
						/>
						<FormControlLabel
							control={<Checkbox checked={formData.enabled} onChange={e => setFormData({ ...formData, enabled: e.target.checked })} />}
							label='Включено'
						/>
					</Box>
				</Box>
			</DialogContent>

			<DialogActions sx={{ px: 3, pb: 2.5 }}>
				<Button onClick={onClose} color='inherit'>
					Отмена
				</Button>
				<Button onClick={handleSubmit} variant='contained' disabled={!isFormValid}>
					{templateId ? 'Сохранить' : 'Создать'}
				</Button>
			</DialogActions>
		</Dialog>
	)
}
