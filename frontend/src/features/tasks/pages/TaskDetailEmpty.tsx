import { Box, Typography, Chip, Stack, Button } from '@mui/material'
import { ArrowLeft, FileQuestion } from 'lucide-mui'
import { useNavigate } from 'react-router'

export default function DetailNotFound() {
	const navigate = useNavigate()

	const handleBack = () => navigate(-1)

	return (
		<Box
			sx={{
				maxWidth: 'xl',
				m: 'auto',
				p: 3,
				textAlign: 'center',
				borderRadius: 3,
				bgcolor: 'background.paper',
				border: '1px solid #e5e7eb',
				boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
			}}
		>
			<Stack
				direction='row'
				spacing={2}
				sx={{
					mb: 2,
					display: 'flex',
					justifyContent: 'center',
					alignItems: 'center',
					gap: 2,
				}}
			>
				<Box
					sx={{
						display: 'inline-flex',
						alignItems: 'center',
						justifyContent: 'center',
						width: 128,
						height: 128,
						borderRadius: '50%',
						bgcolor: 'grey.100',
						mb: 3,
					}}
				>
					<FileQuestion sx={{ fontSize: 76, color: '#9ca3af' }} />
				</Box>

				<Chip
					label='Ошибка 404'
					sx={{
						bgcolor: 'grey.200',
						color: 'text.secondary',
						fontWeight: 600,
						fontSize: '0.875rem',
					}}
				/>
			</Stack>

			<Typography variant='h4' component='h1' sx={{ fontWeight: 'bold', color: 'text.primary', mb: 1.5 }}>
				Заявка не найдена
			</Typography>

			<Typography variant='body2' sx={{ color: 'text.disabled', mb: 4 }}>
				Возможно, она была удалена, перемещена в архив или у вас нет прав на её просмотр
			</Typography>

			<Button
				variant='outlined'
				startIcon={<ArrowLeft sx={{ fontSize: 18 }} />}
				onClick={handleBack}
				sx={{
					color: 'text.primary',
					borderColor: 'grey.300',
					textTransform: 'none', // Отменяет автоматический UPPERCASE в MUI
					fontWeight: 600,
					px: 3,
					py: 1,
					borderRadius: 2, // Скругленные углы в стиле Tailwind rounded-lg
					'&:hover': {
						bgcolor: 'grey.50',
						borderColor: 'grey.400',
					},
				}}
			>
				Вернуться назад
			</Button>
		</Box>
	)
}
