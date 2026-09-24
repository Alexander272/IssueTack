import { useState } from 'react'
import {
	Autocomplete,
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
} from '@mui/material'
import { toast } from 'react-toastify'
import { useFieldArray, useFormContext } from 'react-hook-form'
import { ListTree, Plus, Save, Trash2 } from 'lucide-mui'
import { SectionCard } from './SectionCard'
import type { FormValues } from './types'
import {
	useCreateChecklistTemplateMutation,
	useGetChecklistTemplatesQuery,
	useLazyGetChecklistTemplateItemsQuery,
	useSetChecklistTemplateItemsMutation,
} from '../../modules/checklistTemplates/checklistTemplatesApiSlice'
import type { IChecklistTemplate } from '../../modules/checklistTemplates/types'

type Props = {
	number?: number
}

export const SubtasksCreationSection = ({ number = 3 }: Props) => {
	const { control, register } = useFormContext<FormValues>()
	const { fields, append, remove } = useFieldArray({ control, name: 'subtasks' })

	const [insertOpen, setInsertOpen] = useState(false)
	const [selectedTemplate, setSelectedTemplate] = useState<IChecklistTemplate | null>(null)

	const [saveOpen, setSaveOpen] = useState(false)
	const [templateTitle, setTemplateTitle] = useState('')

	const { data: templatesData, isFetching: isTemplatesLoading } = useGetChecklistTemplatesQuery()
	const [fetchItems] = useLazyGetChecklistTemplateItemsQuery()
	const [createTemplate, { isLoading: isCreating }] = useCreateChecklistTemplateMutation()
	const [setItems, { isLoading: isSavingItems }] = useSetChecklistTemplateItemsMutation()

	const handleInsert = async () => {
		if (!selectedTemplate) return
		const { data } = await fetchItems(selectedTemplate.id)
		if (!data) return

		const items = data.data ?? []
		if (items.length > 0) {
			append(items.map(item => ({ title: item.title, description: item.description })))
			toast.success(`Добавлено подзадач из шаблона: ${items.length}`)
		}
		setSelectedTemplate(null)
		setInsertOpen(false)
	}

	const handleSave = async () => {
		const title = templateTitle.trim()
		if (!title || fields.length === 0) return

		try {
			const result = await createTemplate({ title, description: '' }).unwrap()
			if (result.id) {
				await setItems({
					id: result.id,
					items: fields.map((field, index) => ({
						title: field.title,
						description: field.description,
						sortOrder: index,
					})),
				}).unwrap()
			}
			toast.success('Шаблон сохранён')
			setTemplateTitle('')
			setSaveOpen(false)
		} catch {
			// Ошибка (включая «такое название уже есть») показывается в тосте мутации
		}
	}

	const closeInsert = () => {
		setSelectedTemplate(null)
		setInsertOpen(false)
	}

	const closeSave = () => {
		setTemplateTitle('')
		setSaveOpen(false)
	}

	return (
		<SectionCard number={number} title='Подзадачи' subtitle='Задачи, которые нужно выполнить вместе с заявкой'>
			<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
				{fields.map((field, index) => (
					<Box
						key={field.id}
						sx={{
							display: 'grid',
							gridTemplateColumns: { xs: '1fr' },
							gap: 1,
							p: 1.5,
							border: '1px solid #e5e7eb',
							borderRadius: '10px',
							bgcolor: '#f9fafb',
						}}
					>
						<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
							<TextField
								size='small'
								fullWidth
								placeholder='Заголовок подзадачи'
								{...register(`subtasks.${index}.title`)}
							/>
							<IconButton
								size='small'
								onClick={() => remove(index)}
								sx={{ color: '#9ca3af', '&:hover': { color: 'error.main' } }}
								aria-label='Удалить подзадачу'
							>
								<Trash2 sx={{ fontSize: 18 }} />
							</IconButton>
						</Box>
						<TextField
							size='small'
							multiline
							minRows={2}
							placeholder='Описание подзадачи (необязательно)'
							{...register(`subtasks.${index}.description`)}
						/>
					</Box>
				))}

				<Button
					type='button'
					variant='outlined'
					startIcon={<Plus sx={{ fontSize: 18 }} />}
					onClick={() => append({ title: '', description: '' })}
					sx={{ textTransform: 'none', justifyContent: 'center' }}
				>
					Добавить подзадачу
				</Button>

				<Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
					<Button
						type='button'
						startIcon={<ListTree sx={{ fontSize: 18 }} />}
						onClick={() => setInsertOpen(true)}
						sx={{ textTransform: 'none', justifyContent: 'center' }}
					>
						Вставить из шаблона
					</Button>
					<Button
						type='button'
						startIcon={<Save sx={{ fontSize: 18 }} />}
						onClick={() => setSaveOpen(true)}
						disabled={fields.length === 0}
						sx={{ textTransform: 'none', justifyContent: 'center' }}
					>
						Сохранить как шаблон
					</Button>
				</Stack>

				{fields.length > 0 && (
					<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
						Подзадачи будут созданы вместе с заявкой
					</Typography>
				)}
			</Box>

			<Dialog open={insertOpen} onClose={closeInsert} maxWidth='sm' fullWidth>
				<DialogTitle sx={{ fontSize: '1rem' }}>Вставить подзадачи из шаблона</DialogTitle>
				<DialogContent>
					<Autocomplete
						options={templatesData?.data ?? []}
						getOptionLabel={o => o.title}
						value={selectedTemplate}
						onChange={(_, value) => setSelectedTemplate(value)}
						loading={isTemplatesLoading}
						noOptionsText='Нет шаблонов'
						renderInput={params => <TextField {...params} label='Шаблон' size='small' sx={{ mt: 1 }} />}
					/>
				</DialogContent>
				<DialogActions sx={{ px: 3, pb: 2 }}>
					<Button onClick={closeInsert} sx={{ textTransform: 'none' }}>
						Отмена
					</Button>
					<Button
						variant='contained'
						onClick={handleInsert}
						disabled={!selectedTemplate}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Вставить
					</Button>
				</DialogActions>
			</Dialog>

			<Dialog open={saveOpen} onClose={closeSave} maxWidth='sm' fullWidth>
				<DialogTitle sx={{ fontSize: '1rem' }}>Сохранить подзадачи как шаблон</DialogTitle>
				<DialogContent>
					<TextField
						autoFocus
						fullWidth
						label='Название шаблона'
						value={templateTitle}
						onChange={e => setTemplateTitle(e.target.value)}
						onKeyDown={e => {
							if (e.key === 'Enter') handleSave()
						}}
						sx={{ mt: 1 }}
					/>
				</DialogContent>
				<DialogActions sx={{ px: 3, pb: 2 }}>
					<Button onClick={closeSave} sx={{ textTransform: 'none' }}>
						Отмена
					</Button>
					<Button
						variant='contained'
						onClick={handleSave}
						disabled={!templateTitle.trim() || isCreating || isSavingItems}
						sx={{ textTransform: 'none', boxShadow: 'none', '&:hover': { boxShadow: 'none' } }}
					>
						Сохранить
					</Button>
				</DialogActions>
			</Dialog>
		</SectionCard>
	)
}