import { Alert, Button, Link } from '@mui/material'
import { Link as RouterLink } from 'react-router'
import { Info } from 'lucide-mui'

import type { ITask } from '../../types/task'
import { useDuplicateUnlink } from '../../hooks/useDuplicateUnlink'
import { UnlinkDuplicateDialog } from './UnlinkDuplicateDialog'

interface Props {
	task: ITask
}

/** Баннер над заявкой, которая сама является дублем другой заявки. */
export const DuplicateBanner = ({ task }: Props) => {
	const { target, setTarget, unlinking, confirm } = useDuplicateUnlink()
	const number = task.duplicateOfNumber ?? '—'
	const canUnlink = Boolean(task.access?.canUnlinkDuplicate)

	return (
		<>
			<Alert
				severity='warning'
				role='status'
				icon={<Info sx={{ fontSize: 18 }} />}
				action={
					canUnlink ? (
						<Button
							size='small'
							color='inherit'
							onClick={() => setTarget({ id: task.id, ticketNumber: task.ticketNumber, isSelf: true })}
							sx={{ textTransform: 'none', fontWeight: 600, whiteSpace: 'nowrap' }}
						>
							Отвязать
						</Button>
					) : undefined
				}
				sx={{
					mb: 2,
					borderRadius: '12px',
					alignItems: 'center',
					'& .MuiAlert-message': { fontSize: 13 },
					'& .MuiAlert-action': { p: 0 },
				}}
			>
				Это дубль заявки{' '}
				<Link
					component={RouterLink}
					to={`/tasks/${task.duplicateOf}`}
					underline='hover'
					sx={{ color: 'inherit', fontWeight: 700 }}
				>
					№{number}
				</Link>
				. Все обновления будут приходить в основную.
			</Alert>
			<UnlinkDuplicateDialog
				target={target}
				loading={unlinking}
				onConfirm={confirm}
				onCancel={() => setTarget(null)}
			/>
		</>
	)
}
