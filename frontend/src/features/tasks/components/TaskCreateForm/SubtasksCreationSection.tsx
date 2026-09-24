import { Box, Button, IconButton, TextField, Typography } from '@mui/material'
import { useFieldArray, useFormContext } from 'react-hook-form'
import { Plus, Trash2 } from 'lucide-mui'
import { SectionCard } from './SectionCard'
import type { FormValues } from './types'

type Props = {
	number?: number
}

export const SubtasksCreationSection = ({ number = 3 }: Props) => {
	const { control, register } = useFormContext<FormValues>()
	const { fields, append, remove } = useFieldArray({ control, name: 'subtasks' })

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

				{fields.length > 0 && (
					<Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
						Подзадачи будут созданы вместе с заявкой
					</Typography>
				)}
			</Box>
		</SectionCard>
	)
}