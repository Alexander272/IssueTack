import type { FC } from 'react'
import { Box, Typography } from '@mui/material'
import { keyframes } from '@emotion/react'
import { ChevronRightIcon } from 'lucide-mui'

import type { IUserShort } from '@/features/user/types/user'
import type { ITask } from '@/features/tasks/types/task'
import { ACTIVE_STATUSES, STATUS_MAP } from '@/features/tasks/constants/taskMaps'
import { getShortDate } from '@/utils/date'

type Props = {
	task: ITask
	onClick: (task: ITask) => void
}

// Незавершённые заявки «дышат» точкой статуса — так в списке видно, что заявка
// живая, не читая подпись статуса.
const pulse = keyframes`
	0%, 100% { opacity: 1; }
	50% { opacity: 0.35; }
`

// «Иванов А.» — компактное имя для строки, где места на полное ФИО нет.
const shortName = (user: IUserShort): string => {
	const initial = user.firstName?.[0]
	return [user.lastName, initial ? `${initial}.` : ''].filter(Boolean).join(' ') || user.username
}

// Компактная карточка заявки в блоке «Заявки по оборудованию»: слева иконка статуса,
// в центре номер/тема и строка с датой и участником, справа статус и шеврон. Полная
// карточка из списка задач (TaskCard) для секции модалки слишком крупная.
export const EquipmentTaskItem: FC<Props> = ({ task, onClick }) => {
	const status = STATUS_MAP[task.status]
	const StatusIcon = status.icon
	const isActive = ACTIVE_STATUSES.includes(task.status)

	// Роль участника показываем вместе с именем: у заявки без исполнителя вторая
	// строка иначе молчит о том, кто за неё отвечает.
	const participant = task.assignee ?? task.owner ?? task.creator
	const participantRole = task.assignee ? 'Исполнитель' : task.owner ? 'Заказчик' : 'Автор'

	return (
		<Box
			onClick={() => onClick(task)}
			sx={{
				display: 'flex',
				alignItems: 'center',
				justifyContent: 'space-between',
				gap: 2,
				p: 1.5,
				border: '1px solid #eef2f6',
				borderRadius: '12px',
				bgcolor: '#f8fafc',
				cursor: 'pointer',
				transition: 'all 0.15s ease',
				'&:hover': { bgcolor: '#eff6ff', borderColor: '#bfdbfe' },
				'&:hover .it-task-arrow': { color: '#2563eb' },
			}}
		>
			<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flex: 1, minWidth: 0, overflow: 'hidden' }}>
				<Box
					sx={{
						width: 36,
						height: 36,
						flexShrink: 0,
						borderRadius: '10px',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						bgcolor: status.bgColor,
						color: status.textColor,
					}}
				>
					<StatusIcon sx={{ fontSize: 16 }} />
				</Box>

				<Box sx={{ minWidth: 0 }}>
					<Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
						{task.ticketNumber != null && (
							<Typography
								component='span'
								sx={{ fontSize: 12, fontWeight: 700, color: '#64748b', flexShrink: 0 }}
							>
								№{task.ticketNumber}
							</Typography>
						)}
						<Typography
							sx={{
								fontSize: 14,
								fontWeight: 600,
								color: '#1e293b',
								overflow: 'hidden',
								textOverflow: 'ellipsis',
								whiteSpace: 'nowrap',
							}}
						>
							{task.title}
						</Typography>
					</Box>
					<Typography
						sx={{
							mt: 0.25,
							fontSize: 12,
							color: '#94a3b8',
							overflow: 'hidden',
							textOverflow: 'ellipsis',
							whiteSpace: 'nowrap',
						}}
					>
						От {getShortDate(task.createdAt)} • {participantRole}: {shortName(participant)}
					</Typography>
				</Box>
			</Box>

			<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, flexShrink: 0 }}>
				<Box
					sx={{
						display: 'inline-flex',
						alignItems: 'center',
						gap: 0.75,
						px: 1.25,
						py: 0.5,
						borderRadius: '999px',
						bgcolor: status.bgColor,
						color: status.textColor,
					}}
				>
					<Box
						component='span'
						sx={{
							width: 6,
							height: 6,
							borderRadius: '50%',
							bgcolor: status.textColor,
							...(isActive && { animation: `${pulse} 1.6s ease-in-out infinite` }),
						}}
					/>
					<Typography sx={{ fontSize: 11, fontWeight: 600, lineHeight: 1 }}>{status.label}</Typography>
				</Box>

				<ChevronRightIcon className='it-task-arrow' sx={{ fontSize: 14, color: '#cbd5e1' }} />
			</Box>
		</Box>
	)
}
