import type { ITask } from '../../types/task'
import { TaskRow } from './TaskRow'

interface Props {
	tasks: ITask[]
	onTaskClick: (task: ITask) => void
}

export const TaskTableBody = ({ tasks, onTaskClick }: Props) => (
	<>
		{tasks.map(task => (
			<TaskRow key={task.id} task={task} onClick={onTaskClick} />
		))}
	</>
)
