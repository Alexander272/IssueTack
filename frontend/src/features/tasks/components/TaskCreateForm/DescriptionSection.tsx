import { Box, TextField, Typography } from '@mui/material'
import { Controller, useFormContext } from 'react-hook-form'
import { SectionCard } from './SectionCard'
import { FileDropZone } from '../../modules/attachments/FileDropZone'
import { fieldLabelSx } from './styles'
import type { FormValues } from './types'

type Props = {
	files?: File[]
	onFilesChange?: (files: File[]) => void
	/** Только для чтения — скрывает вложения и блокирует поля */
	readOnly?: boolean
}

export const DescriptionSection = ({ files, onFilesChange, readOnly = false }: Props) => {
	const { control } = useFormContext<FormValues>()

	return (
		<SectionCard
			number={2}
			title={'Заголовок и описание'}
			subtitle={readOnly ? undefined : 'Чем подробнее вы опишете проблему, тем быстрее её решат'}
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
					rules={{
						required: 'Обязательное поле',
						validate: value => (value ?? '').trim().length > 0 || 'Заголовок не может состоять из пробелов',
					}}
					render={({ field, fieldState }) => (
						<Box>
							<TextField
								{...field}
								fullWidth
								size='small'
								placeholder={readOnly ? undefined : 'Например: Не работает принтер в кабинете 305'}
								disabled={readOnly}
								error={!readOnly && Boolean(fieldState.error)}
								helperText={!readOnly && fieldState.error?.message}
								slotProps={{ htmlInput: { maxLength: 150 } }}
							/>
							{!readOnly && (
								<Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
									<Typography sx={{ fontSize: '0.6875rem', color: '#9ca3af', mt: 0.25 }}>
										{(field.value ?? '').length} / 150
									</Typography>
								</Box>
							)}
						</Box>
					)}
				/>
			</Box>

			<Box>
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
							disabled={readOnly}
							placeholder={
								readOnly
									? undefined
									: 'Опишите, что произошло:&#10;• Что вы делали перед проблемой?&#10;• Что именно не работает?&#10;• Появляются ли ошибки?&#10;• Когда это началось?'
							}
						/>
					)}
				/>
			</Box>

			{!readOnly && files && onFilesChange && (
				<Box sx={{ mt: 1.5 }}>
					<Typography variant='caption' sx={fieldLabelSx}>
						Вложения
					</Typography>
					<FileDropZone files={files} onChange={onFilesChange} />
				</Box>
			)}
		</SectionCard>
	)
}
