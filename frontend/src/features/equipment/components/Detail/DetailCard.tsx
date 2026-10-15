import type { FC, ReactNode } from 'react'
import { Box, Typography } from '@mui/material'

import { sectionLabelSx } from './consts'

type Props = {
	title?: string
	action?: ReactNode
	// divided — разделитель под шапкой (в макете он есть у характеристик).
	divided?: boolean
	children: ReactNode
}

// Карточка секции вкладки «Обзор»: белая, скруглённая, с тонкой рамкой и мягкой
// тенью. Заголовок и действие — в одной строке; карточка без title — просто рамка
// (например, блок держателя рисует подпись внутри).
export const DetailCard: FC<Props> = ({ title, action, divided, children }) => (
	<Box
		sx={{
			bgcolor: '#fff',
			border: '1px solid #e9eef5',
			borderRadius: '16px',
			boxShadow: '0 1px 2px rgba(16, 24, 40, 0.04)',
			p: 2.5,
		}}
	>
		{title && (
			<Box
				sx={{
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'space-between',
					gap: 2,
					mb: 1.5,
					pb: divided ? 1.5 : 0,
					borderBottom: divided ? '1px solid #f1f5f9' : 'none',
				}}
			>
				<Typography sx={sectionLabelSx}>{title}</Typography>
				{action}
			</Box>
		)}
		{children}
	</Box>
)
