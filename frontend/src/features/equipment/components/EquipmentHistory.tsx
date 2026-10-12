import { useState, type FC } from 'react'
import { Box, Tab, Table, TableBody, TableCell, TableHead, TableRow, Tabs, Typography } from '@mui/material'

import type { IEquipment } from '../types/equipment'
import { getEquipmentEventLabel, getEquipmentStatus } from '../types/equipment'
import {
	useGetEquipmentAssignmentsQuery,
	useGetEquipmentCompositionHistoryQuery,
	useGetEquipmentEventsQuery,
} from '../equipmentApiSlice'

type Props = {
	unit: IEquipment
}

const fmtDateTime = (d: string | null) => (d ? new Date(d).toLocaleString('ru-RU') : '—')

const emptyCell = (colSpan: number, text: string) => (
	<TableRow>
		<TableCell colSpan={colSpan} align='center' sx={{ py: 3, color: '#6b7280' }}>
			{text}
		</TableCell>
	</TableRow>
)

export const EquipmentHistory: FC<Props> = ({ unit }) => {
	const [tab, setTab] = useState(0)

	const { data: assignmentsData } = useGetEquipmentAssignmentsQuery(unit.id)
	const { data: eventsData } = useGetEquipmentEventsQuery({ id: unit.id, limit: 100 })
	const { data: compositionData } = useGetEquipmentCompositionHistoryQuery(unit.id, { skip: !unit.isKit })

	const assignments = assignmentsData?.data ?? []
	const events = eventsData?.data ?? []
	const composition = compositionData?.data ?? []

	return (
		<Box sx={{ border: '1px solid #eef2f6', borderRadius: '16px', bgcolor: '#fff', overflow: 'hidden' }}>
			<Tabs
				value={tab}
				onChange={(_e, value) => setTab(value)}
				sx={{ px: 2, borderBottom: '1px solid #eef2f6' }}
			>
				<Tab label={`Выдачи (${assignments.length})`} />
				<Tab label={`События (${events.length})`} />
				{unit.isKit && <Tab label={`История состава (${composition.length})`} />}
			</Tabs>

			{tab === 0 && (
				<Table size='small'>
					<TableHead>
						<TableRow sx={{ bgcolor: '#fafbfc' }}>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Выдан</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Возвращён</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Держатель</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Локация</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Комментарий</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{assignments.length === 0
							? emptyCell(5, 'Выдач пока не было')
							: assignments.map(a => (
									<TableRow key={a.id}>
										<TableCell sx={{ whiteSpace: 'nowrap' }}>{fmtDateTime(a.assignedAt)}</TableCell>
										<TableCell sx={{ whiteSpace: 'nowrap' }}>
											{a.unassignedAt ? fmtDateTime(a.unassignedAt) : 'текущая'}
										</TableCell>
										<TableCell>{a.holderName || '—'}</TableCell>
										<TableCell>{a.location || '—'}</TableCell>
										<TableCell>{a.comment || '—'}</TableCell>
									</TableRow>
								))}
					</TableBody>
				</Table>
			)}

			{tab === 1 && (
				<Table size='small'>
					<TableHead>
						<TableRow sx={{ bgcolor: '#fafbfc' }}>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Дата</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Событие</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Комментарий</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{events.length === 0
							? emptyCell(3, 'Событий пока нет')
							: events.map(e => (
									<TableRow key={e.id}>
										<TableCell sx={{ whiteSpace: 'nowrap' }}>{fmtDateTime(e.createdAt)}</TableCell>
										<TableCell>{getEquipmentEventLabel(e.eventType)}</TableCell>
										<TableCell>{e.comment || '—'}</TableCell>
									</TableRow>
								))}
					</TableBody>
				</Table>
			)}

			{unit.isKit && tab === 2 && (
				<Table size='small'>
					<TableHead>
						<TableRow sx={{ bgcolor: '#fafbfc' }}>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Компонент</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>С</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>По</TableCell>
							<TableCell sx={{ fontWeight: 600, color: '#6b7280' }}>Причина</TableCell>
						</TableRow>
					</TableHead>
					<TableBody>
						{composition.length === 0
							? emptyCell(4, 'История состава пуста')
							: composition.map(c => (
									<TableRow key={c.id}>
										<TableCell sx={{ fontFamily: 'monospace', fontSize: 12 }}>
											{c.componentId.slice(0, 8)}
										</TableCell>
										<TableCell sx={{ whiteSpace: 'nowrap' }}>{fmtDateTime(c.startedAt)}</TableCell>
										<TableCell sx={{ whiteSpace: 'nowrap' }}>
											{c.endedAt ? fmtDateTime(c.endedAt) : 'текущий'}
										</TableCell>
										<TableCell>{c.reason || '—'}</TableCell>
									</TableRow>
								))}
					</TableBody>
				</Table>
			)}

			<Box sx={{ px: 2, py: 1 }}>
				<Typography variant='caption' sx={{ color: '#9ca3af' }}>
					Статус единицы: {getEquipmentStatus(unit.status).label}
				</Typography>
			</Box>
		</Box>
	)
}
