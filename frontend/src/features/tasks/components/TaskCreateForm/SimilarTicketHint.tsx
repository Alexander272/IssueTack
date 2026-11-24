import { useState } from 'react'
import { Link as RouterLink } from 'react-router'
import { Box, Button, Link, Tooltip, Typography } from '@mui/material'
import { Info } from 'lucide-mui'

import type { ITask } from '../../types/task'
import { useGetTasksQuery } from '../../tasksApiSlice'
import { STATUS_MAP } from '../../constants/taskMaps'
import { getShortDate } from '@/utils/date'
import { useDebounce } from '@/hooks/useDebounce'

const MIN_QUERY_LENGTH = 5
// В основную можно превратить только активную заявку — LinkDuplicate не примет
// closed/cancelled, поэтому такие совпадения в подсказке бесполезны.
const PRIMARY_STATUSES = ['open', 'in_progress', 'pending', 'on_hold', 'resolved']

interface Props {
	title: string
}

// Визуальная заготовка будущего сценария: при новом обращении с похожей
// проблемой пользователь сможет привязать его к предыдущей заявке. Пока
// подсказка только показывает кандидата и ведёт в его карточку; связывать
// здесь нечего до успешного создания заявки.
export const SimilarTicketHint = ({ title }: Props) => {
	const query = useDebounce(title.trim(), 400)
	const [dismissedFor, setDismissedFor] = useState<string | null>(null)

	// readable: подсказка показывается только по заявкам, которые пользователь
	// вправе открыть; дефолтный скоуп списка не нашёл бы его собственную заявку
	// из группы, в которую он не входит.
	const { data } = useGetTasksQuery(
		{ search: query, limit: 5, mode: 'readable' },
		{ skip: query.length < MIN_QUERY_LENGTH },
	)

	const suggestion: ITask | undefined = (data?.data ?? []).find(
		candidate => !candidate.duplicateOf && PRIMARY_STATUSES.includes(candidate.status),
	)

	if (query.length < MIN_QUERY_LENGTH || dismissedFor === query || !suggestion) return null

	const status = STATUS_MAP[suggestion.status]

	return (
		<Box
			role='note'
			sx={{
				display: 'flex',
				gap: 1.5,
				p: 2,
				borderRadius: '12px',
				bgcolor: '#f8fafc',
				border: '1px solid #e6e8ec',
			}}
		>
			<Info sx={{ fontSize: 18, color: '#1a56c4', flex: 'none', mt: '2px' }} />
			<Box sx={{ minWidth: 0 }}>
				<Typography sx={{ fontSize: 14, fontWeight: 600, color: '#1f2430' }}>
					Похожая заявка уже есть: №{suggestion.ticketNumber ?? '—'}
				</Typography>
				<Typography sx={{ fontSize: 13, color: '#6b7280' }}>
					{suggestion.title} · {status?.label ?? suggestion.status} · {getShortDate(suggestion.createdAt)}
				</Typography>
				<Typography sx={{ fontSize: 12, color: '#6b7280', mt: 1 }}>
					Скоро здесь можно будет привязать новое обращение к предыдущему.
				</Typography>
				<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: '4px 16px', mt: 1 }}>
					<Tooltip title='Привязка похожих обращений появится позже'>
						<span>
							<Button
								size='small'
								disabled
								sx={{ textTransform: 'none', color: '#1a56c4', fontWeight: 600, fontSize: 13, px: 1 }}
							>
								Проблема вернулась, связать
							</Button>
						</span>
					</Tooltip>
					<Button
						size='small'
						onClick={() => setDismissedFor(query)}
						sx={{ textTransform: 'none', color: '#6b7280', fontSize: 13, px: 1 }}
					>
						Нет, это другое
					</Button>
					<Link
						component={RouterLink}
						to={`/tasks/${suggestion.id}`}
						underline='hover'
						sx={{ fontSize: 13, color: '#1a56c4', alignSelf: 'center' }}
					>
						Открыть заявку
					</Link>
				</Box>
			</Box>
		</Box>
	)
}
