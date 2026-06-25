import { type FC } from 'react'
import {
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
import { getSmartDate } from '@/utils/date'
import { TaskPriorityBadge } from '@/features/tasks/components/TaskPriorityBadge'
import { StatusBadge } from '@/features/access/components/StatusBadge'
import { EditIcon } from '@/components/Icons/EditIcon'
import { EyeIcon } from '@/components/Icons/EyeIcon'
import { GroupsIcon } from '@/components/Icons/GroupsIcon'

const statusLabel = (active: boolean) => (active ? 'Активна' : 'Неактивна')

type Props = {
	categories: ICategory[]
	groupsMap: Map<string, string>
	onView: (cat: ICategory) => void
	onEdit: (cat: ICategory) => void
}

export const CategoryTable: FC<Props> = ({ categories, groupsMap, onView, onEdit }) => {
	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden', overflowX: 'auto' }}
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
										<GroupsIcon sx={{ fontSize: 16, mr: 0.5, fill: '#1d4ed8' }} />
										{groupsMap.get(cat.groupId) || 'Неизвестная группа'}
									</Typography>
								</TableCell>

								<TableCell sx={{ px: 3 }}>
									<TaskPriorityBadge priority={cat.priority} />
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
											<IconButton
												onClick={() => onView(cat)}
												size='large'
												sx={{ ':hover': { svg: { fill: '#f59e0b' } } }}
											>
												<EyeIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
										<Tooltip title='Редактировать'>
											<IconButton
												onClick={() => onEdit(cat)}
												size='large'
												sx={{ ':hover': { svg: { fill: '#3b82f6' } } }}
											>
												<EditIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
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
