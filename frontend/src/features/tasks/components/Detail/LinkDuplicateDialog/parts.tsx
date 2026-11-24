import { memo, type FC, type ReactNode } from 'react'
import { Box, Button, Chip, CircularProgress, List, ListItem, ListItemButton, Typography } from '@mui/material'

import type { ITask, TicketStatus } from '../../../types/task'
import { STATUS_MAP } from '../../../constants/taskMaps'
import { personName, ticketNumber, unavailableReason } from './utils'

export const StatusPill: FC<{ status: TicketStatus }> = ({ status }) => {
	const info = STATUS_MAP[status]
	return (
		<Chip
			size='small'
			label={info?.label ?? status}
			sx={{
				flex: 'none',
				height: 18,
				borderRadius: '10px',
				fontWeight: 600,
				bgcolor: info?.bgColor ?? 'action.hover',
				color: info?.textColor ?? 'text.secondary',
				'& .MuiChip-label': { px: 1, fontSize: '0.75rem', lineHeight: '18px' },
			}}
		/>
	)
}

export const FieldLabel: FC<{ children: ReactNode; htmlFor?: string }> = ({ children, htmlFor }) => (
	<Typography
		component='label'
		htmlFor={htmlFor}
		variant='caption'
		color='text.secondary'
		sx={{ display: 'block', mb: 0.75, fontWeight: 600 }}
	>
		{children}
	</Typography>
)

type CandidateListProps = {
	candidates: ITask[]
	isFetching: boolean
	task: ITask
	onPick: (candidate: ITask) => void
}

/**
 * memo: при вводе в поле поиска пропсы стабильны (task, candidates из RTKQ, onPick = setPicked),
 * поэтому строки не перерисовываются на каждый keystroke.
 */
export const CandidateList = memo(function CandidateList({ candidates, isFetching, task, onPick }: CandidateListProps) {
	return (
		<List
			role='listbox'
			aria-label='Результаты поиска'
			sx={{
				mt: 1,
				p: 0.5,
				maxHeight: 210,
				overflow: 'auto',
				border: '1px solid',
				borderColor: 'divider',
				borderRadius: '8px',
			}}
		>
			{isFetching && (
				<ListItem sx={{ justifyContent: 'center', py: 1.5 }}>
					<CircularProgress size={18} />
				</ListItem>
			)}
			{!isFetching && candidates.length === 0 && (
				<ListItem sx={{ justifyContent: 'center', color: 'text.secondary', fontSize: 13 }}>
					Ничего не найдено
				</ListItem>
			)}
			{candidates.map(candidate => {
				const reason = unavailableReason(candidate, task)
				return (
					<ListItemButton
						key={candidate.id}
						role='option'
						aria-selected={false}
						aria-disabled={Boolean(reason)}
						disabled={Boolean(reason)}
						onClick={() => onPick(candidate)}
						sx={{ borderRadius: '6px', gap: 1, py: 1, alignItems: 'flex-start', display: 'block' }}
					>
						<Box>
							<Typography
								component='span'
								variant='body2'
								color='primary'
								sx={{ fontWeight: 700, mr: 1 }}
							>
								№{ticketNumber(candidate)}
							</Typography>
							<Typography component='span' variant='body2'>
								{candidate.title}
							</Typography>
						</Box>
						<Box
							sx={{
								display: 'flex',
								justifyContent: 'space-between',
								alignItems: 'center',
								gap: 1,
								mt: 0.25,
							}}
						>
							<Typography variant='caption' color='text.secondary' sx={{ minWidth: 0 }}>
								{reason ?? personName(candidate.creator)}
							</Typography>
							<StatusPill status={candidate.status} />
						</Box>
					</ListItemButton>
				)
			})}
		</List>
	)
})

export const PickedPrimaryCard: FC<{ picked: ITask; onChange: () => void }> = ({ picked, onChange }) => (
	<Box
		sx={{
			display: 'flex',
			gap: 1.25,
			alignItems: 'flex-start',
			p: 1.5,
			border: '1px solid',
			borderColor: 'primary.main',
			borderRadius: '10px',
			bgcolor: 'primary.light',
		}}
	>
		<Box sx={{ minWidth: 0, flex: 1 }}>
			<Box sx={{ display: 'block', alignItems: 'baseline' }}>
				<Typography
					component='span'
					variant='body2'
					color='primary'
					sx={{ flex: 'none', fontWeight: 700, mr: 1 }}
				>
					№{ticketNumber(picked)}
				</Typography>
				<Typography component='span' variant='body2' sx={{ minWidth: 0 }}>
					{picked.title}
				</Typography>
			</Box>

			<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 0.25 }}>
				<Typography variant='caption' color='text.secondary'>
					{personName(picked.creator)}
				</Typography>
				<StatusPill status={picked.status} />
			</Box>
		</Box>
		<Button size='small' onClick={onChange} sx={{ textTransform: 'none', fontWeight: 600, flex: 'none' }}>
			Изменить
		</Button>
	</Box>
)
