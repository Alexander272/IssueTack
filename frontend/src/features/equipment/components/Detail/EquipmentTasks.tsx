import { Box } from '@mui/material'

import { Loader } from '@/components/Fallback/Loader'
import { TaskCardList } from '@/features/tasks/components/Table/TaskCardList'
import { useGetTasksQuery } from '@/features/tasks/tasksApiSlice'

// Заявки, привязанные к единице оборудования.
export const EquipmentTasks = ({ unitId }: { unitId: string }) => {
	const { data, isLoading, isError } = useGetTasksQuery({ equipmentId: unitId, limit: 50 })

	if (isLoading) return <Loader />
	if (isError || !data?.data || data.data.length === 0) {
		return (
			<Box sx={{ p: 2, color: '#6b7280', fontSize: 14, border: '1px dashed #e5e7eb', borderRadius: '12px' }}>
				По этому оборудованию заявок нет
			</Box>
		)
	}

	return <TaskCardList tasks={data.data} groupBy='category' groupEnabled={false} onTaskClick={() => {}} />
}
