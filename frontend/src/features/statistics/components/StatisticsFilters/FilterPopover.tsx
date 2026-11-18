import { useState, type FC } from 'react'
import { Box, Button, Popover as MuiPopover, Typography } from '@mui/material'

import { FilterSelect } from './FilterSelect'
import type { FilterPopoverBodyProps, PopoverProps } from './types'

const sectionSx = { fontSize: 12, fontWeight: 600, color: '#57606a', textTransform: 'uppercase' as const, mb: 0.75 }
const sectionBoxSx = { mb: 2, pb: 2, borderBottom: '1px solid #eaeef2' }

const FilterPopoverBody: FC<FilterPopoverBodyProps> = ({
	initial,
	onClose,
	isManager,
	assigneeOptions,
	categoryOptions,
	groupOptions,
	siteOptions,
	onApply,
}) => {
	const [local, setLocal] = useState(() => ({
		assigneeId: initial.assigneeId,
		categoryId: initial.categoryId,
		groupId: initial.groupId,
		siteId: initial.siteId,
	}))

	const update = <K extends keyof typeof local>(key: K, value: (typeof local)[K]) => {
		setLocal(prev => ({ ...prev, [key]: value }))
	}

	const handleApply = () => {
		onApply({ ...local })
		onClose()
	}

	const handleReset = () => {
		setLocal({ assigneeId: [], categoryId: [], groupId: [], siteId: [] })
	}

	return (
		<Box sx={{ width: 380, p: 2 }}>
			{isManager && (
				<Box sx={sectionBoxSx}>
					<Typography sx={sectionSx}>Исполнитель</Typography>
					<FilterSelect
						options={assigneeOptions}
						values={local.assigneeId}
						onChange={ids => update('assigneeId', ids)}
					/>
				</Box>
			)}

			<Box sx={sectionBoxSx}>
				<Typography sx={sectionSx}>Категория</Typography>
				<FilterSelect
					options={categoryOptions}
					values={local.categoryId}
					onChange={ids => update('categoryId', ids)}
				/>
			</Box>

			{isManager && (
				<Box sx={sectionBoxSx}>
					<Typography sx={sectionSx}>Группа</Typography>
					<FilterSelect
						options={groupOptions}
						values={local.groupId}
						onChange={ids => update('groupId', ids)}
					/>
				</Box>
			)}

			<Box sx={{ mb: 2 }}>
				<Typography sx={sectionSx}>Площадка</Typography>
				<FilterSelect options={siteOptions} values={local.siteId} onChange={ids => update('siteId', ids)} />
			</Box>

			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					gap: 1,
					mt: 2,
					pt: 2,
					borderTop: '1px solid #eaeef2',
				}}
			>
				<Button
					onClick={handleReset}
					sx={{
						color: '#2f81f7',
						textTransform: 'none',
						p: 0,
						minWidth: 'auto',
						'&:hover': { textDecoration: 'underline', bgcolor: 'transparent' },
					}}
				>
					Сбросить всё
				</Button>
				<Button
					variant='contained'
					onClick={handleApply}
					sx={{ textTransform: 'none', borderRadius: '6px' }}
				>
					Применить
				</Button>
			</Box>
		</Box>
	)
}

export const FilterPopover: FC<PopoverProps> = ({
	open,
	anchorEl,
	onClose,
	initial,
	onApply,
	isManager,
	assigneeOptions,
	categoryOptions,
	groupOptions,
	siteOptions,
}) => {
	return (
		<MuiPopover
			open={open}
			anchorEl={anchorEl}
			onClose={onClose}
			anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
			transformOrigin={{ vertical: 'top', horizontal: 'right' }}
			slotProps={{
				paper: {
					sx: { borderRadius: '10px', boxShadow: '0 10px 30px rgba(0,0,0,0.12)', mt: '0.5' },
				},
			}}
		>
			{open && (
				<FilterPopoverBody
					initial={initial}
					onClose={onClose}
					isManager={isManager}
					assigneeOptions={assigneeOptions}
					categoryOptions={categoryOptions}
					groupOptions={groupOptions}
					siteOptions={siteOptions}
					onApply={onApply}
				/>
			)}
		</MuiPopover>
	)
}
