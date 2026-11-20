import { memo } from 'react'
import { Button, Checkbox, FormControlLabel, Stack, Typography } from '@mui/material'
import { ChevronsDownIcon, InfoIcon } from 'lucide-mui'

import { ACCENT, buttonSx } from './tableStyles'

type Props = {
	shownCount: number
	totalCount: number
	showHidden: boolean
	onShowHiddenChange: (value: boolean) => void
	allOpen: boolean
	onToggleAll: (expand: boolean) => void
	isFiltering: boolean
}

export const CategoryTableToolbar = memo(function CategoryTableToolbar({
	shownCount,
	totalCount,
	showHidden,
	onShowHiddenChange,
	allOpen,
	onToggleAll,
	isFiltering,
}: Props) {
	return (
		<>
			<Stack direction='row' spacing={1.25} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 1.5 }}>
				{/* <Button
					variant='outlined'
					color='inherit'
					onClick={onToggleAll}
					sx={{ height: 36, textTransform: 'none', borderColor: '#d1d5db' }}
				>
					{allOpen ? 'Свернуть все' : 'Развернуть все'}
				</Button> */}
				<Button
					variant='outlined'
					color='inherit'
					aria-label='Развернуть всё'
					disabled={allOpen}
					onClick={() => onToggleAll(false)}
					sx={buttonSx}
				>
					<ChevronsDownIcon sx={{ fontSize: 16, mr: 1 }} />
					Развернуть всё
				</Button>

				<Button
					variant='outlined'
					color='inherit'
					aria-label='Свернуть всё'
					disabled={!allOpen}
					onClick={() => onToggleAll(true)}
					sx={buttonSx}
				>
					<ChevronsDownIcon sx={{ fontSize: 16, mr: 1, transform: 'rotate(180deg)' }} />
					Свернуть всё
				</Button>

				<FormControlLabel
					control={
						<Checkbox
							size='small'
							checked={showHidden}
							onChange={e => onShowHiddenChange(e.target.checked)}
						/>
					}
					label={
						<Typography sx={{ fontSize: '0.875rem', color: 'text.secondary' }}>
							Показывать скрытые
						</Typography>
					}
				/>
				<Typography sx={{ ml: 'auto', color: 'text.secondary', fontSize: '0.8125rem' }}>
					Показано {shownCount} из {totalCount}
				</Typography>
			</Stack>

			{isFiltering && (
				<Stack
					direction='row'
					spacing={1}
					sx={{
						alignItems: 'center',
						bgcolor: '#eceefc',
						color: ACCENT,
						px: 1.5,
						py: 1,
						borderRadius: 2,
						mb: 1.5,
					}}
				>
					<InfoIcon sx={{ fontSize: 16 }} />
					<Typography sx={{ fontSize: '0.875rem' }}>
						При поиске и фильтрах перетаскивание отключено. Сбросьте их, чтобы менять порядок.
					</Typography>
				</Stack>
			)}
		</>
	)
})
