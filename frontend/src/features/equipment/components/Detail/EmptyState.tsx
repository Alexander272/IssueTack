import type { ElementType, FC } from 'react'
import { Box, Typography } from '@mui/material'

type Props = {
	icon: ElementType
	title: string
	hint?: string
}

// Пустое состояние секции: иконка в круге, заголовок и подсказка — как в макете
// у блока «Заявки по оборудованию».
export const EmptyState: FC<Props> = ({ icon: Icon, title, hint }) => (
	<Box
		sx={{
			py: 1.5,
			px: 2,
			display: 'flex',
			flexDirection: 'column',
			alignItems: 'center',
			gap: 1,
			textAlign: 'center',
			bgcolor: '#f8fafc',
			border: '1px dashed #e2e8f0',
			borderRadius: '12px',
		}}
	>
		<Box
			sx={{
				width: 40,
				height: 40,
				borderRadius: '50%',
				bgcolor: '#eef2f7',
				color: '#94a3b8',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'center',
			}}
		>
			<Icon sx={{ fontSize: 20 }} />
		</Box>
		<Typography sx={{ fontSize: 14, fontWeight: 500, color: '#64748b' }}>{title}</Typography>
		{hint && <Typography sx={{ fontSize: 12, color: '#94a3b8' }}>{hint}</Typography>}
	</Box>
)
