import { Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'
import { SectionCard } from './SectionCard'
import { FileDropZone } from './FileDropZone'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'

type Props = {
	files: File[]
	onFilesChange: (files: File[]) => void
}

export const DescriptionSection = ({ files, onFilesChange }: Props) => {
	const { control } = useFormContext<FormValues>()

	return (
		<SectionCard
			number={2}
			title='Описание проблемы'
			subtitle='Чем подробнее вы опишете проблему, тем быстрее её решат'
		>
			<Box>
				<Typography variant='caption' sx={fieldLabelSx}>
					Заголовок{' '}
					<Typography component='span' color='error'>
						*
					</Typography>
				</Typography>
				<Controller
					control={control}
					name='title'
					rules={{ required: 'Обязательное поле' }}
					render={({ field, fieldState }) => (
						<Box>
							<TextField
								{...field}
								fullWidth
								size='small'
								placeholder='Например: Не работает принтер в кабинете 305'
								error={Boolean(fieldState.error)}
								helperText={fieldState.error?.message}
								slotProps={{ htmlInput: { maxLength: 150 } }}
							/>
							<Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
								<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', mt: 0.25 }}>
									{(field.value ?? '').length} / 150
								</Typography>
							</Box>
						</Box>
					)}
				/>
			</Box>

			<Box sx={{ mt: 3 }}>
				<Typography variant='caption' sx={fieldLabelSx}>
					Подробное описание
				</Typography>
				<Controller
					control={control}
					name='description'
					render={({ field }) => (
						<TextField
							{...field}
							fullWidth
							size='small'
							multiline
							minRows={6}
							placeholder='Опишите, что произошло:&#10;• Что вы делали перед проблемой?&#10;• Что именно не работает?&#10;• Появляются ли ошибки?&#10;• Когда это началось?'
						/>
					)}
				/>
			</Box>

			<Box sx={{ mt: 3 }}>
				<Typography variant='caption' sx={fieldLabelSx}>
					Вложения
				</Typography>
				<FileDropZone files={files} onChange={onFilesChange} />
			</Box>
		</SectionCard>
	)
}
