import { TableRow, TableCell, Typography, Box } from '@mui/material'

import type { ITask } from '../types/task'
import { TaskStatusBadge } from './TaskStatusBadge'
import { TaskPriorityBadge } from './TaskPriorityBadge'
import { TaskAssignmentChip } from './TaskAssignmentChip'
import { TaskProgressBar } from './TaskProgressBar'

interface Props {
  task: ITask
  onClick: (task: ITask) => void
}

export const TaskRow = ({ task, onClick }: Props) => {
  const subtaskProgress = task.subtasks
    ? {
        done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
        total: task.subtasks.length,
      }
    : { done: 0, total: 0 }

  const requesterLabel = task.creator
    ? `${task.creator.fullName}${task.owner ? ` (${task.owner.fullName})` : ''}`
    : '—'

  const deadlineFormatted = task.dueDate || '—'

  return (
    <TableRow
      hover
      onClick={() => onClick(task)}
      sx={{
        cursor: 'pointer',
        borderBottom: '1px solid #f3f4f6',
        '&:hover': { bgcolor: '#f8fafc' },
        '&:last-child td': { border: 0 },
      }}
    >
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          <Typography sx={{ fontWeight: 600, color: '#1f2937', fontSize: '0.875rem' }}>
            #{task.ticketNumber ?? task.id.slice(0, 4)} {task.title}
          </Typography>
          <Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', mt: 0.25 }}>
            {task.description.length > 60 ? `${task.description.slice(0, 60)}...` : task.description}
          </Typography>
        </Box>
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <Typography sx={{ fontSize: '0.875rem', color: '#374151' }}>{requesterLabel}</Typography>
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <Box
          sx={{
            display: 'inline-flex',
            px: 1.25,
            py: 0.25,
            borderRadius: '999px',
            fontSize: '0.75rem',
            fontWeight: 500,
            bgcolor: '#f3f4f6',
            color: '#374151',
          }}
        >
          {task.category.name}
        </Box>
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <TaskAssignmentChip assignee={task.assignee} group={task.group} />
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <TaskStatusBadge status={task.status} />
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <TaskPriorityBadge priority={task.priority} />
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          <Typography sx={{ fontSize: '0.875rem', color: '#4b5563' }}>
            {deadlineFormatted}
          </Typography>
          {task.closedAt && (
            <Typography sx={{ fontSize: '0.75rem', color: '#059669' }}>
              Закрыта: {task.closedAt}
            </Typography>
          )}
        </Box>
      </TableCell>
      <TableCell sx={{ px: 3, py: 2.5 }} align='right'>
        <TaskProgressBar done={subtaskProgress.done} total={subtaskProgress.total} />
      </TableCell>
    </TableRow>
  )
}
