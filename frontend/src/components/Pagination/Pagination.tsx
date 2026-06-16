import type { FC } from 'react'
import { Box, Stack, type SxProps, type Theme, useTheme } from '@mui/material'

import { LeftArrowIcon } from '../Icons/LeftArrowIcon'
import { Button } from './Button'

type Props = {
	page: number
	totalPages: number
	onClick: (page: number) => void
	sx?: SxProps<Theme>
}

const defaultPosition = 36

export const Pagination: FC<Props> = ({ page, totalPages, onClick, sx }) => {
	const { palette } = useTheme()

	const boundaryCount = 1
	const siblingCount = 1

	const range = (start: number, end: number) => {
		const length = end - start + 1
		return Array.from({ length }, (_, i) => start + i)
	}

	const startPages = range(1, Math.min(boundaryCount, totalPages))
	const endPages = range(Math.max(totalPages - boundaryCount + 1, boundaryCount + 1), totalPages)

	const siblingsStart = Math.max(
		Math.min(
			// Natural start
			page - siblingCount,
			// Lower boundary when page is high
			totalPages - boundaryCount - siblingCount * 2 - 1,
		),
		// Greater than startPages
		boundaryCount + 2,
	)

	const siblingsEnd = Math.min(
		Math.max(
			// Natural end
			page + siblingCount,
			// Upper boundary when page is low
			boundaryCount + siblingCount * 2 + 2,
		),
		// Less than endPages
		endPages.length > 0 ? endPages[0] - 2 : totalPages - 1,
	)

	const itemList = [
		...startPages,

		// Start ellipsis
		...(siblingsStart > boundaryCount + 2
			? ['start-ellipsis']
			: boundaryCount + 1 < totalPages - boundaryCount
				? [boundaryCount + 1]
				: []),

		// Sibling pages
		...range(siblingsStart, siblingsEnd),

		// End ellipsis
		...(siblingsEnd < totalPages - boundaryCount - 1
			? ['end-ellipsis']
			: totalPages - boundaryCount > boundaryCount
				? [totalPages - boundaryCount]
				: []),

		...endPages,
	]

	const selectHandler = (item: number | string, index?: number) => () => {
		if (typeof item === 'number' && item <= totalPages && item >= 1) onClick(item)
		else if (item == 'start-ellipsis') onClick((itemList[(index || -1) + 1] as number) - 1)
		else if (item == 'end-ellipsis') onClick((itemList[(index || 1) - 1] as number) + 1)
	}

	return (
		<Stack direction={'row'} sx={{ borderRadius: 3, position: 'relative', backgroundColor: '#f4f4f5', ...sx }}>
			<Box
				component={'span'}
				translate='yes'
				sx={{
					position: 'absolute',
					height: '100%',
					width: 36,
					display: 'flex',
					justifyContent: 'center',
					alignItems: 'center',
					bgcolor: palette.primary.main,
					borderRadius: 3,
					color: '#fff',
					zIndex: 5,
					transition: 'all .3s ease-in-out',
					transform: `translateX(${(itemList.findIndex(i => i == page) + 1) * defaultPosition}px) scale(1)`,
				}}
			>
				{page}
			</Box>

			<Button onClick={selectHandler(page - 1)}>
				<LeftArrowIcon fontSize={12} fill={page == 1 ? palette.action.disabled : palette.primary.main} />
			</Button>

			{itemList.map((item, i) => (
				<Button key={item} onClick={selectHandler(item, i)}>
					{typeof item === 'number' ? item : '...'}
				</Button>
			))}

			<Button onClick={selectHandler(page + 1)}>
				<LeftArrowIcon
					fontSize={12}
					transform='rotate(180deg)'
					fill={page == totalPages ? palette.action.disabled : palette.primary.main}
				/>
			</Button>
		</Stack>
	)
}
