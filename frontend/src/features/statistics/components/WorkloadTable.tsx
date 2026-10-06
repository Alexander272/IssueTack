import type { FC } from 'react'
import {
	Box,
	LinearProgress,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
} from '@mui/material'

import type { IWorkloadBucket } from '../types/statistics'

interface Props {
	workload: IWorkloadBucket[]
	isLoading: boolean
}

export const WorkloadTable: FC<Props> = ({ workload, isLoading }) => {
	const max = workload.reduce((acc, item) => Math.max(acc, item.active), 0) || 1

	return (
		<Paper sx={{ p: 3, border: '1px solid #eee', borderRadius: 2 }} elevation={0}>
			<Typography variant='h6' sx={{ fontWeight: 600, mb: 2 }}>
				Нагрузка исполнителей
			</Typography>
			{workload.length === 0 && !isLoading ? (
				<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
					<Typography variant='body2'>Нет данных за выбранный период</Typography>
				</Box>
			) : (
				<TableContainer>
					<Table size='small'>
						<TableHead>
							<TableRow>
								<TableCell>Исполнитель</TableCell>
								<TableCell align='right' sx={{ width: 100 }}>
									Активные
								</TableCell>
								<TableCell align='right' sx={{ width: 100 }}>
									Всего
								</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{workload.map(item => (
								<TableRow key={item.userId} hover>
									<TableCell>
										<Typography variant='body2' sx={{ mb: 0.5 }}>
											{item.name}
										</Typography>
										<LinearProgress
											variant='determinate'
											value={(item.active / max) * 100}
											sx={{ height: 4, borderRadius: 2 }}
										/>
									</TableCell>
									<TableCell align='right'>{item.active}</TableCell>
									<TableCell align='right'>{item.total}</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</TableContainer>
			)}
		</Paper>
	)
}