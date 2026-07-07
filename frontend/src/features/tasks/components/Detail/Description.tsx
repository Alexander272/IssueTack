import { Box, Button, Typography } from '@mui/material'
import { PenSquareIcon, AlignLeft } from 'lucide-mui'

interface Props {
	description: string
}

export const Description = ({ description }: Props) => {
	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box
				sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', px: 2.5, py: 2, borderBottom: '1px solid #e5e7eb' }}
			>
				<Typography sx={{ fontWeight: 700, color: '#1f2937', fontSize: '0.9375rem', display: 'flex', alignItems: 'center', gap: 1 }}>
					<AlignLeft sx={{ fontSize: 16 }} />
					Описание
				</Typography>
				<Button
					size='small'
					sx={{
						textTransform: 'none',
						fontSize: '0.75rem',
						color: '#9ca3af',
						gap: 0.5,
						'&:hover': { color: 'primary.main' },
					}}
				>
					<PenSquareIcon sx={{ fontSize: 13 }} />
					Редактировать
				</Button>
			</Box>
			<Box sx={{ p: 2.5 }}>
				<Typography sx={{ color: '#374151', fontSize: '0.875rem', lineHeight: 1.7, whiteSpace: 'pre-wrap' }}>
					{description}
				</Typography>
			</Box>
		</Box>
	)
}
