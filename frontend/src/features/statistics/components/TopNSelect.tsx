import type { FC } from 'react'
import { Box, MenuItem, Select, Typography } from '@mui/material'

// Сколько сущностей показывать в донат-диаграмме. Максимум 20 + «Прочие»: сотни
// секторов нечитаемы, поэтому опции «Все» нет. 8 — дефолт, выбранный в плане статистики.
export const DEFAULT_TOP_N = 8

const OPTIONS = [5, 8, 10, 20]

interface Props {
	value: number
	onChange: (value: number) => void
}

export const TopNSelect: FC<Props> = ({ value, onChange }) => {
	return (
		<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
			<Typography
				variant='body2'
				sx={{
					color: 'text.secondary',
					fontWeight: 500,
					userSelect: 'none',
				}}
			>
				Показывать:
			</Typography>

			<Select
				value={String(value)}
				onChange={e => onChange(Number(e.target.value))}
				variant='standard'
				disableUnderline
				// В MUI v9 используем слот-пропсы для настройки вложенного Menu и Paper
				MenuProps={{
					slotProps: {
						paper: {
							sx: {
								borderRadius: 2, // Мягкие скругления углов меню
								boxShadow: '0px 4px 20px rgba(0, 0, 0, 0.08)', // Легкая современная тень
								mt: 0.5,
							},
						},
					},
				}}
				sx={{
					minWidth: 80,
					fontWeight: 600,
					color: 'text.primary',
					fontSize: '0.875rem',
					cursor: 'pointer',
					'& .MuiSelect-select': {
						py: 0.5,
						px: 1,
						borderRadius: 1,
						backgroundColor: 'action.hover',
						'&:focus': {
							backgroundColor: 'action.selected',
						},
					},
				}}
			>
				{OPTIONS.map(option => (
					<MenuItem
						key={option}
						value={String(option)}
						sx={{
							fontSize: '0.875rem',
							borderRadius: 1,
							mx: 0.5,
							my: 0.25,
							'&.Mui-selected': {
								fontWeight: 600,
							},
						}}
					>
						Топ {option}
					</MenuItem>
				))}
			</Select>
		</Box>
	)
}
