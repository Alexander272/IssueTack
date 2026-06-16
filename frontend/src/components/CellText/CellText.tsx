import { Tooltip, Typography } from '@mui/material'
import { type FC, useEffect, useRef, useState } from 'react'

type Props = {
	value: string
	color?: string
	align?: 'right' | 'left' | 'center' | 'inherit' | 'justify'
}

export const CellText: FC<Props> = ({ value, align = 'center', color }) => {
	const [isOverflow, setIsOverflow] = useState<boolean>(false)
	const overflowingText = useRef<HTMLParagraphElement | null>(null)

	const checkOverflow = (el: HTMLParagraphElement | null): boolean => {
		if (el) return el.offsetHeight < el.scrollHeight || el.offsetWidth < el.scrollWidth
		return false
	}

	useEffect(() => {
		setIsOverflow(checkOverflow(overflowingText.current))
	}, [])

	const text = (
		<Typography
			ref={overflowingText}
			align={align}
			sx={{
				p: '6px 6px',
				mr: 0.5,
				overflow: 'hidden',
				textOverflow: 'ellipsis',
				whiteSpace: 'nowrap',
				color: color,
			}}
		>
			{value}
		</Typography>
	)

	if (isOverflow) return <Tooltip title={value}>{text}</Tooltip>
	return text
}
