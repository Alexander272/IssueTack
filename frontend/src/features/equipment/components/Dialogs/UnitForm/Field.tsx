import type { FC, ReactNode } from 'react'
import { Box, Typography } from '@mui/material'

// Подпись поля: мелкий полужирный заголовок над контролом, со звёздочкой
// обязательности при needed.
export const FieldLabel: FC<{ children: ReactNode; required?: boolean }> = ({ children, required }) => (
	<Typography variant='caption' sx={{ fontWeight: 600, mb: 0.5, display: 'block' }}>
		{children}
		{required && (
			<Typography component='span' color='error'>
				{' '}
				*
			</Typography>
		)}
	</Typography>
)

// Обёртка «подпись + контрол» — чтобы поля формы не дублировали разметку заголовка.
export const Field: FC<{ label: string; required?: boolean; children: ReactNode }> = ({
	label,
	required,
	children,
}) => (
	<Box>
		<FieldLabel required={required}>{label}</FieldLabel>
		{children}
	</Box>
)
