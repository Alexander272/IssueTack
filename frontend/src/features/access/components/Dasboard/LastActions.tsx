import { Fragment, useState, type FC } from 'react'
import {
	Box,
	Typography,
	Paper,
	Skeleton,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Chip,
	Collapse,
	IconButton,
	type SxProps,
} from '@mui/material'

import { getSmartDate } from '@/utils/date'
import { stringToHSLA } from '@/utils/colors'
import { useGetAuditLogsQuery } from '../../auditApiSlice'
import { UnfoldLessIcon } from '@/components/Icons/UnfoldLessIcon'
import { UnfoldMoreIcon } from '@/components/Icons/UnfoldMoreIcon'

export const LastActions = () => {
	const { data: logs, isFetching: isFetchingLogs } = useGetAuditLogsQuery(null)
	const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())

	const toggleRow = (index: number) => {
		setExpandedRows(prev => {
			const next = new Set(prev)
			if (next.has(index)) {
				next.delete(index)
			} else {
				next.add(index)
			}
			return next
		})
	}

	const formatJson = (value: unknown) => {
		if (!value) return '-'
		try {
			return JSON.stringify(value, null, 2)
		} catch {
			return String(value)
		}
	}

	return (
		<>
			<Typography variant='subtitle1' sx={{ fontSize: '15px', fontWeight: 'bold', mb: 2 }}>
				Последние действия
			</Typography>

			<TableContainer component={Paper} elevation={0} sx={{ border: '1px solid #eee', borderRadius: 2 }}>
				<Table>
					<TableHead>
						<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
							<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>Пользователь</TableCell>
							<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
								Действие
							</TableCell>
							<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
								Тип объекта
							</TableCell>
							<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
								Объект
							</TableCell>
							<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
								ID
							</TableCell>
							<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>
								Дата
							</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{isFetchingLogs ? (
							<>
								<TableRow>
									<TableCell colSpan={6}>
										<Skeleton variant='rounded' width='100%' height={40} />
									</TableCell>
								</TableRow>
								<TableRow>
									<TableCell colSpan={6}>
										<Skeleton variant='rounded' width='100%' height={40} />
									</TableCell>
								</TableRow>
							</>
						) : null}

						{!logs?.data?.length ? (
							<TableRow>
								<TableCell colSpan={4} align='center' sx={{ py: 3, color: 'text.secondary' }}>
									Действий пока нет.
								</TableCell>
							</TableRow>
						) : null}

						{logs?.data.map((log, i) => (
							<Fragment key={i}>
								<TableRow hover onClick={() => toggleRow(i)}>
									<TableCell>
										<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
											<IconButton size='small'>
												{expandedRows.has(i) ? (
													<UnfoldLessIcon fontSize='small' />
												) : (
													<UnfoldMoreIcon fontSize='small' />
												)}
											</IconButton>
											<Box>
												<Typography variant='body2' sx={{ fontWeight: 500 }}>
													{log.changedByName}
												</Typography>
												<Typography
													variant='caption'
													color='text.secondary'
													sx={{ display: 'block' }}
												>
													{log.changedBy}
												</Typography>
											</Box>
										</Box>
									</TableCell>
									<TableCell align='center'>
										<ActionChip action={log.action} />
									</TableCell>
									<TableCell align='center'>
										<ActionChip action={log.entityType} />
									</TableCell>
									<TableCell align='center'>{log.entity || '-'}</TableCell>
									<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '13px' }}>
										{log.entityId || '-'}
									</TableCell>
									<TableCell align='center' sx={{ color: 'text.secondary', fontSize: '13px' }}>
										{getSmartDate(log.createdAt)}
									</TableCell>
								</TableRow>

								<TableRow>
									<TableCell
										colSpan={6}
										sx={{
											py: 0,
											borderBottom: expandedRows.has(i) ? '1px solid #eee' : 'none',
										}}
									>
										<Collapse in={expandedRows.has(i)} timeout='auto' unmountOnExit>
											<Box sx={{ py: 2, display: 'flex', gap: 4 }}>
												<Box sx={{ flex: 1 }}>
													<Typography
														variant='caption'
														color='text.secondary'
														sx={{ display: 'block', mb: 0.5 }}
													>
														Старые значения
													</Typography>
													<Box
														component='pre'
														sx={{
															fontSize: '12px',
															background: '#f5f5f5',
															p: 1,
															borderRadius: 1,
															overflow: 'auto',
															maxHeight: 200,
														}}
													>
														{formatJson(log.oldValues)}
													</Box>
												</Box>
												<Box sx={{ flex: 1 }}>
													<Typography
														variant='caption'
														color='text.secondary'
														sx={{ display: 'block', mb: 0.5 }}
													>
														Новые значения
													</Typography>
													<Box
														component='pre'
														sx={{
															fontSize: '12px',
															background: '#f5f5f5',
															p: 1,
															borderRadius: 1,
															overflow: 'auto',
															maxHeight: 200,
														}}
													>
														{formatJson(log.newValues)}
													</Box>
												</Box>
											</Box>
										</Collapse>
									</TableCell>
								</TableRow>
							</Fragment>
						))}
					</TableBody>
				</Table>
			</TableContainer>
		</>
	)
}

const ActionChip: FC<{ action: string; sx?: SxProps }> = ({ action, sx }) => {
	const colors = stringToHSLA(action)

	return (
		<Chip
			label={action}
			size={'small'}
			sx={{
				backgroundColor: colors.bg,
				color: colors.text,
				border: `1px solid ${colors.border}`,
				borderRadius: '6px',
				fontSize: '0.75rem',
				height: '20px',
				fontWeight: 500,
				...sx,
			}}
		/>
	)
}
