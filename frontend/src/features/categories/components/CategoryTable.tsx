import { type FC } from 'react'
import {
	Box,
	IconButton,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Tooltip,
	Typography,
} from '@mui/material'

import type { ICategory } from '../types/category'
import type { Priority } from '@/features/tasks/types/task'
import { getSmartDate } from '@/utils/date'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { StatusBadge } from '@/features/access/components/StatusBadge'
import { ModifyIcon } from '@/components/Icons/ModifyIcon'
import { VisibleIcon } from '@/components/Icons/VisibleIcon'
import { TimesIcon } from '@/components/Icons/TimesIcon'

const PRIORITY_COLORS: Record<Priority, { bg: string; text: string; border: string }> = {
	low: { bg: '#d1fae5', text: '#065f46', border: '#a7f3d0' },
	medium: { bg: '#fef3c7', text: '#b45309', border: '#fde68a' },
	high: { bg: '#fee2e2', text: '#b91c1c', border: '#fecaca' },
	urgent: { bg: '#fee2e2', text: '#b91c1c', border: '#fecaca' },
}

const statusLabel = (active: boolean) => (active ? 'Активна' : 'Неактивна')

type Props = {
	categories: ICategory[]
	groupsMap: Map<string, string>
	onView: (cat: ICategory) => void
	onEdit: (cat: ICategory) => void
	onToggle: (cat: ICategory) => void
}

export const CategoryTable: FC<Props> = ({ categories, groupsMap, onView, onEdit, onToggle }) => {
	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden' }}
		>
			<Table sx={{ minWidth: 900 }}>
				<TableHead>
					<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
						<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
							Название
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
							Описание
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Группа-владелец
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Приоритет
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Статус
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Обновлено
						</TableCell>
						<TableCell align='right' sx={{ p: 0, width: 120 }}></TableCell>
					</TableRow>
				</TableHead>

				<TableBody sx={{ '& tr:not(:last-child)': { borderBottom: '1px solid #f3f4f6' } }}>
					{categories.map(cat => {
						const priorityColor = PRIORITY_COLORS[cat.priority]
						const priorityInfo = PRIORITY_MAP[cat.priority]

						return (
							<TableRow
								key={cat.id}
								hover
								sx={{
									cursor: 'pointer',
									'&:hover': { bgcolor: '#fafafa' },
									opacity: cat.isActive ? 1 : 0.6,
								}}
							>
								<TableCell sx={{ py: 2, px: 4 }}>
									<Typography sx={{ fontWeight: 600, color: '#111827' }}>{cat.name}</Typography>
								</TableCell>

								<TableCell sx={{ py: 2, px: 4 }}>
									<Typography
										sx={{
											fontSize: '0.875rem',
											color: '#6b7280',
											maxWidth: 280,
											overflow: 'hidden',
											textOverflow: 'ellipsis',
											whiteSpace: 'nowrap',
										}}
									>
										{cat.description}
									</Typography>
								</TableCell>

								<TableCell sx={{ px: 3 }}>
									<Typography
										sx={{
											display: 'inline-flex',
											alignItems: 'center',
											gap: 0.75,
											px: 1.5,
											py: 0.5,
											borderRadius: '6px',
											fontSize: '0.75rem',
											fontWeight: 500,
											bgcolor: '#eff6ff',
											color: '#1d4ed8',
											border: '1px solid #bfdbfe',
										}}
									>
										{groupsMap.get(cat.groupId) || 'Неизвестная группа'}
									</Typography>
								</TableCell>

								<TableCell sx={{ px: 3 }}>
									<Box
										sx={{
											display: 'inline-flex',
											alignItems: 'center',
											gap: 0.75,
											px: 1.5,
											py: 0.25,
											borderRadius: '999px',
											fontSize: '0.75rem',
											fontWeight: 500,
											bgcolor: priorityColor.bg,
											color: priorityColor.text,
											border: `1px solid ${priorityColor.border}`,
										}}
									>
										{priorityInfo.label}
									</Box>
								</TableCell>

								<TableCell sx={{ px: 3 }}>
									<StatusBadge active={cat.isActive} label={statusLabel(cat.isActive)} />
								</TableCell>

								<TableCell sx={{ px: 3, color: '#6b7280', fontSize: '0.8rem' }}>
									{getSmartDate(cat.updatedAt)}
								</TableCell>

								<TableCell align='center' sx={{ p: 0, pr: 1 }}>
									<Stack direction='row' sx={{ justifyContent: 'flex-end' }}>
										<Tooltip title='Просмотр'>
											<IconButton onClick={() => onView(cat)} size='small'>
												<VisibleIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
										<Tooltip title='Редактировать'>
											<IconButton onClick={() => onEdit(cat)} size='small'>
												<ModifyIcon sx={{ fontSize: 18 }} />
											</IconButton>
										</Tooltip>
										<Tooltip title={cat.isActive ? 'Деактивировать' : 'Активировать'}>
											<IconButton onClick={() => onToggle(cat)} size='small'>
												{cat.isActive ? (
													<TimesIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
												) : (
													<ModifyIcon sx={{ fontSize: 18, fill: '#10b981' }} />
												)}
											</IconButton>
										</Tooltip>
									</Stack>
								</TableCell>
							</TableRow>
						)
					})}
					{!categories.length ? (
						<TableRow>
							<TableCell colSpan={7} align='center' sx={{ py: 3, color: 'text.secondary' }}>
								Категории не найдены.
							</TableCell>
						</TableRow>
					) : null}
				</TableBody>
			</Table>
		</TableContainer>
	)
}
