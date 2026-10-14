import type { FC, ReactNode } from 'react'
import {
	Box,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
} from '@mui/material'
import { InboxIcon } from 'lucide-mui'

import { Loader } from '@/components/Fallback/Loader'
import { pluralize } from '@/utils/plural'

// Число колонок таблицы оборудования — для colSpan строк загрузки и пустоты.
export const EQUIPMENT_TABLE_COLUMNS = 6

type Props = {
	isLoading: boolean
	total: number
	empty: boolean
	children: ReactNode
}

// Каркас таблицы оборудования: шапка, состояния загрузки/пустоты и итог под
// таблицей. Тело собирает потребитель — дерево или плоский список.
export const EquipmentTable: FC<Props> = ({ isLoading, total, empty, children }) => (
	<>
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: 3, border: '1px solid #f3f4f6', overflow: 'hidden', overflowX: 'auto' }}
		>
			<Table sx={{ tableLayout: 'fixed', minWidth: 1040 }}>
				<TableHead>
					<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
						<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem' }}>Название</TableCell>
						<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 170 }}>Тип</TableCell>
						<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 200 }}>Держатель</TableCell>
						<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 180 }}>
							Серийный номер
						</TableCell>
						<TableCell sx={{ color: 'text.secondary', fontSize: '0.875rem', width: 150 }}>Статус</TableCell>
						<TableCell sx={{ width: 44 }} />
					</TableRow>
				</TableHead>
				<TableBody>
					{isLoading && empty ? (
						<TableRow>
							<TableCell colSpan={EQUIPMENT_TABLE_COLUMNS} align='center' sx={{ py: 8 }}>
								<Loader />
							</TableCell>
						</TableRow>
					) : empty ? (
						<TableRow>
							<TableCell colSpan={EQUIPMENT_TABLE_COLUMNS} align='center' sx={{ py: 8 }}>
								<Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 0.5 }}>
									<InboxIcon sx={{ fontSize: 36, color: '#d1d5db', mb: 0.5 }} />
									<Typography sx={{ fontSize: '0.875rem', fontWeight: 500, color: '#6b7280' }}>
										Ничего не найдено
									</Typography>
									<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
										Попробуйте изменить фильтры или сбросить поиск
									</Typography>
								</Box>
							</TableCell>
						</TableRow>
					) : (
						children
					)}
				</TableBody>
			</Table>
		</TableContainer>

		{/* Итоги под таблицей — как на странице заявок. */}
		<Typography sx={{ mt: 1.5, px: 1, fontSize: '0.875rem', color: '#6b7280' }}>
			Всего: {total} {pluralize(total, ['позиция', 'позиции', 'позиций'])}
		</Typography>
	</>
)
