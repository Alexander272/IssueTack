import { Box, Typography } from '@mui/material'
import { Info, Layers, Building2, Flag, Calendar, Clock, Hash } from 'lucide-mui'

import type { ITask } from '../../types/task'
import {
	PRIORITY_MAP,
	ACTIVE_STATUSES,
	DUE_DATE_COLORS,
	DUE_DATE_ICONS,
	DUE_DATE_DEFAULT_ICON,
} from '../../constants/taskMaps'
import { getDueDateTone, getSmartDate } from '@/utils/date'
import { useIsRoot } from '@/features/access/utils/can'

interface Props {
	task: ITask
}

export const Meta = ({ task }: Props) => {
	const priorityInfo = PRIORITY_MAP[task.priority]
	const isRoot = useIsRoot()
	const dueTone = ACTIVE_STATUSES.includes(task.status) ? getDueDateTone(task.dueDate) : null
	const DueDateIcon = dueTone ? DUE_DATE_ICONS[dueTone] : DUE_DATE_DEFAULT_ICON

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ px: 2.5, py: 1.5, borderBottom: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
				<Typography
					sx={{
						fontSize: '0.75rem',
						fontWeight: 700,
						color: '#374151',
						textTransform: 'uppercase',
						letterSpacing: '0.05em',
						display: 'flex',
						alignItems: 'center',
						gap: 0.75,
					}}
				>
					<Info sx={{ fontSize: 14 }} />
					Детали
				</Typography>
			</Box>
			<Box sx={{ p: 2.5, display: 'flex', flexDirection: 'column', gap: 2 }}>
				<MetaRow
					label='Категория'
					icon={<Layers sx={{ fontSize: 14 }} />}
					value={
						task.category ? (
							<Box
								sx={{
									display: 'inline-flex',
									alignItems: 'center',
									px: 1.5,
									py: 0.25,
									borderRadius: '999px',
									fontSize: '0.75rem',
									fontWeight: 500,
									bgcolor: '#f3e8ff',
									color: '#6b21a8',
								}}
							>
								{task.category.name}
							</Box>
						) : (
							'—'
						)
					}
				/>
				<MetaRow label='Площадка' icon={<Building2 sx={{ fontSize: 14 }} />} value={task.site?.name || '—'} />
				<MetaRow
					label='Приоритет'
					icon={<Flag sx={{ fontSize: 14 }} />}
					value={
						<Box
							sx={{
								display: 'inline-flex',
								alignItems: 'center',
								gap: 0.5,
								px: 1.5,
								py: 0.25,
								borderRadius: '999px',
								fontSize: '0.75rem',
								fontWeight: 500,
								bgcolor: priorityInfo.bgColor,
								color: priorityInfo.textColor,
							}}
						>
							{priorityInfo.label}
						</Box>
					}
				/>
				<Box sx={{ borderTop: '1px solid #e5e7eb', pt: 2 }} />
				<MetaRow
					label='Создана'
					icon={<Calendar sx={{ fontSize: 14 }} />}
					value={getSmartDate(task.createdAt)}
				/>
				<MetaRow
					label='Срок'
					icon={<DueDateIcon sx={{ fontSize: 14, color: dueTone ? DUE_DATE_COLORS[dueTone] : undefined }} />}
					value={
						<Typography
							sx={{
								fontSize: '0.75rem',
								color: dueTone ? DUE_DATE_COLORS[dueTone] : '#374151',
								fontWeight: dueTone && dueTone !== 'soon' ? 600 : 500,
								textAlign: 'right',
							}}
						>
							{getSmartDate(task.dueDate || '')}
						</Typography>
					}
				/>
				<MetaRow
					label='Обновлена'
					icon={<Clock sx={{ fontSize: 14 }} />}
					value={getSmartDate(task.updatedAt)}
				/>
				{isRoot && (
					<MetaRow
						label='ID'
						icon={<Hash sx={{ fontSize: 14 }} />}
						value={
							<Typography
								component='code'
								sx={{
									fontSize: '0.6875rem',
									bgcolor: '#f3f4f6',
									px: 0.75,
									py: 0.25,
									borderRadius: '4px',
									color: '#6b7280',
								}}
							>
								{task.id}
							</Typography>
						}
					/>
				)}
			</Box>
		</Box>
	)
}

const MetaRow = ({ label, icon, value }: { label: string; icon?: React.ReactNode; value: React.ReactNode }) => (
	<Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 1 }}>
		<Typography
			sx={{
				fontSize: '0.75rem',
				color: '#9ca3af',
				flexShrink: 0,
				display: 'flex',
				alignItems: 'center',
				gap: 0.5,
				verticalAlign: 'middle',
			}}
		>
			{icon} {label}
		</Typography>
		<Box sx={{ textAlign: 'right', fontSize: '0.8125rem', color: '#374151' }}>{value}</Box>
	</Box>
)
