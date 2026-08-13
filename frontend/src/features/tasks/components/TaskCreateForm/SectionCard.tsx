import type { ReactNode } from 'react'
import { Box, Typography } from '@mui/material'

type Props = {
	number: number
	title: string
	subtitle?: string
	children: ReactNode
}

export const SectionCard = ({ number, title, subtitle, children }: Props) => (
	<Box
		sx={{
			bgcolor: '#fff',
			borderRadius: '16px',
			border: '1px solid #e5e7eb',
			boxShadow: '0 1px 2px rgba(0,0,0,0.04)',
			overflow: 'hidden',
		}}
	>
		<Box
			sx={{
				px: 2.5,
				py: 1.75,
				borderBottom: '1px solid #e5e7eb',
				bgcolor: '#f9fafb',
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				gap: 2,
			}}
		>
			<Box>
				<Typography component='div' sx={{ fontWeight: 700, color: '#1f2937', display: 'flex', alignItems: 'center', gap: 1 }}>
					<Box
						sx={{
							width: 24,
							height: 24,
							borderRadius: '50%',
							bgcolor: 'primary.main',
							color: '#fff',
							fontSize: '0.75rem',
							fontWeight: 700,
							display: 'flex',
							alignItems: 'center',
							justifyContent: 'center',
							flexShrink: 0,
						}}
					>
						{number}
					</Box>
					{title}
				</Typography>
				{subtitle && <Typography sx={{ fontSize: '0.75rem', color: '#6b7280', mt: 0.5 }}>{subtitle}</Typography>}
			</Box>
		</Box>
		<Box sx={{ p: 2.5 }}>{children}</Box>
	</Box>
)
