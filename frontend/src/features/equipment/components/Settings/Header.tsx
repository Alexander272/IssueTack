import type { FC } from 'react'
import { Box, Divider, IconButton, Tooltip, Typography } from '@mui/material'
import { ArrowLeftIcon } from 'lucide-mui'
import { useNavigate } from 'react-router'

export const Header: FC = () => {
	const navigate = useNavigate()

	return (
		<Box
			sx={{
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				pb: 1,
				pt: 2,
				borderBottom: '1px solid #e5e7eb',
				gap: 2,
			}}
		>
			<Tooltip title='К списку оборудования'>
				<IconButton
					onClick={() => navigate(-1)}
					sx={{ color: '#6b7280', '&:hover': { svg: { color: 'primary.main' } } }}
				>
					<ArrowLeftIcon sx={{ fontSize: 20 }} />
				</IconButton>
			</Tooltip>

			<Divider flexItem orientation='vertical' />

			<Box sx={{ flexGrow: 1, minWidth: 0, flexBasis: { xs: 0, sm: 'auto' } }}>
				<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
					Настройки оборудования
				</Typography>
				<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
					Типы, шаблоны комплектов
				</Typography>
			</Box>
		</Box>
	)
}
