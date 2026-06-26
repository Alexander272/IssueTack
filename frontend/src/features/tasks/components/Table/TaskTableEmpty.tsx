import { TableCell, TableRow } from '@mui/material'

interface Props {
	columnsCount: number
}

export const TaskTableEmpty = ({ columnsCount }: Props) => (
	<TableRow>
		<TableCell colSpan={columnsCount} align='center' sx={{ py: 3, color: 'text.secondary' }}>
			Нет задач по выбранным фильтрам
		</TableCell>
	</TableRow>
)
