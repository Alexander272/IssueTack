import React, { useState, type KeyboardEvent } from 'react'
import { Box, Button, Chip, TextField, Typography } from '@mui/material'

interface KeywordsFieldProps {
	value?: string[]
	onChange: (tags: string[]) => void
	label?: string
	placeholder?: string
	helperText?: string
	error?: boolean
}

export const KeywordsField: React.FC<KeywordsFieldProps> = ({
	value = [],
	onChange,
	label = 'Ключевые слова',
	placeholder = 'Введите слово и нажмите Enter',
	helperText = 'Подкатегория ставится автоматически, если ключ есть в заголовке или описании новой заявки (можно вводить через запятую)',
	error = false,
}) => {
	const [inputValue, setInputValue] = useState('')

	const addTags = () => {
		if (!inputValue.trim()) return

		const newTags = inputValue
			.split(',')
			.map(t => t.trim())
			.filter(t => t.length > 0 && !value.includes(t))

		if (newTags.length > 0) {
			onChange([...value, ...newTags])
		}
		setInputValue('')
	}

	const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
		if (e.key === 'Enter') {
			e.preventDefault()
			addTags()
		}
	}

	const removeTag = (indexToRemove: number) => {
		onChange(value.filter((_, idx) => idx !== indexToRemove))
	}

	return (
		<Box>
			{label && (
				<Typography
					component='label'
					htmlFor='keywords-input'
					sx={{
						display: 'block',
						fontSize: '13px',
						fontWeight: 500,
						mb: '6px',
						color: error ? 'error.main' : '#374151',
					}}
				>
					{label}
				</Typography>
			)}

			{/* Поле ввода и кнопка */}
			<Box sx={{ display: 'flex', gap: 1 }}>
				<TextField
					fullWidth
					error={error}
					value={inputValue}
					onChange={e => setInputValue(e.target.value)}
					onKeyDown={handleKeyDown}
					placeholder={placeholder}
				/>
				<Button
					variant='outlined'
					color='inherit'
					onClick={addTags}
					sx={{
						flexShrink: 0,
						height: 40,
						borderColor: '#c4c4c4',
						textTransform: 'none' as const,
						whiteSpace: 'nowrap' as const,
						width: { xs: '100%', sm: 'auto' },
					}}
				>
					Добавить
				</Button>
			</Box>

			{/* Контейнер тегов */}
			<Box
				sx={{
					display: 'flex',
					flexWrap: 'wrap',
					alignItems: 'center',
					gap: '6px',
					mt: 1,
				}}
			>
				{value.length === 0 ? (
					<Typography sx={{ color: '#9ca3af', fontSize: '13px' }}>Теги появятся здесь</Typography>
				) : (
					value.map((tag, index) => (
						<Chip
							key={`${tag}-${index}`}
							label={tag}
							size='small'
							onDelete={() => removeTag(index)}
							sx={{
								bgcolor: '#e0e7ff',
								color: '#312e81',
								border: '1px solid #a5b4fc',
								borderRadius: '6px',
								fontSize: '13px',
								'& .MuiChip-deleteIcon': {
									color: '#312e81',
									opacity: 0.6,
									fontSize: '16px',
									'&:hover': {
										color: '#312e81',
										opacity: 1,
									},
								},
							}}
						/>
					))
				)}
			</Box>

			{helperText && (
				<Typography
					sx={{
						fontSize: '12px',
						color: error ? 'error.main' : '#6b7280',
						mt: 0.5,
						mb: 1,
					}}
				>
					{helperText}
				</Typography>
			)}
		</Box>
	)
}
