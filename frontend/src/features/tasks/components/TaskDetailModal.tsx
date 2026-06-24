import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Box,
  Select,
  MenuItem,
  FormControl,
  Grid,
} from '@mui/material'

import type { ITask, TicketStatus } from '../types/task'
import { TaskPriorityBadge } from './TaskPriorityBadge'
import { STATUS_MAP } from '../constants/taskMaps'

interface Props {
  open: boolean
  task: ITask | null
  onClose: () => void
  onStatusChange: (taskId: string, status: TicketStatus) => void
  onSubtaskStatusChange: (taskId: string, subtaskId: string, status: TicketStatus) => void
}

const SUBTASK_STATUS_OPTIONS: { value: TicketStatus; label: string }[] = [
  { value: 'open', label: 'Открыта' },
  { value: 'in_progress', label: 'В работе' },
  { value: 'closed', label: 'Выполнена' },
]

export const TaskDetailModal = ({ open, task, onClose, onStatusChange, onSubtaskStatusChange }: Props) => {
  if (!task) return null

  const subtaskProgress = task.subtasks
    ? {
        done: task.subtasks.filter(s => s.status === 'closed' || s.status === 'resolved').length,
        total: task.subtasks.length,
      }
    : { done: 0, total: 0 }

  const assignmentText = task.assignee
    ? `👤 ${task.assignee.fullName}`
    : task.group
      ? `👥 Группа: ${task.group.name}`
      : '—'

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth='md'
      fullWidth
      slotProps={{
        backdrop: {
          sx: { bgcolor: 'rgba(0,0,0,0.4)', backdropFilter: 'blur(2px)' },
        },
        paper: {
          sx: { borderRadius: '16px', maxHeight: '90vh' },
        },
      }}
    >
      <DialogTitle sx={{ px: 3, py: 2.5, borderBottom: '1px solid #e5e7eb' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <Box
              sx={{
                fontSize: '0.75rem',
                fontFamily: 'monospace',
                bgcolor: '#f3f4f6',
                px: 1.5,
                py: 0.5,
                borderRadius: '999px',
                color: '#6b7280',
                fontWeight: 500,
              }}
            >
              #{task.ticketNumber ?? task.id.slice(0, 4)}
            </Box>
            <TaskPriorityBadge priority={task.priority} />
          </Box>
          <Button
            onClick={onClose}
            sx={{ minWidth: 36, minHeight: 36, p: 0, color: '#9ca3af', fontSize: '1.25rem' }}
          >
            ✕
          </Button>
        </Box>
      </DialogTitle>

      <DialogContent sx={{ px: 3, py: 2.5 }}>
        <Box sx={{ mb: 3 }}>
          <Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937', mb: 0.5 }}>
            {task.title}
          </Typography>
          <Typography sx={{ color: '#6b7280', fontSize: '0.875rem' }}>{task.description}</Typography>
        </Box>

        <Grid container spacing={2} sx={{ bgcolor: '#f9fafb', p: 2, borderRadius: '12px', mb: 3 }}>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography sx={{ color: '#6b7280', fontSize: '0.875rem', display: 'inline' }}>
              Заказчик:{' '}
            </Typography>
            <Typography sx={{ fontWeight: 500, fontSize: '0.875rem', display: 'inline' }}>
              {task.creator ? `${task.creator.fullName}${task.owner ? ` (${task.owner.fullName})` : ''}` : '—'}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography sx={{ color: '#6b7280', fontSize: '0.875rem', display: 'inline' }}>
              Назначено:{' '}
            </Typography>
            <Typography sx={{ fontWeight: 500, fontSize: '0.875rem', display: 'inline' }}>
              {assignmentText}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography sx={{ color: '#6b7280', fontSize: '0.875rem', display: 'inline' }}>
              Категория:{' '}
            </Typography>
            <Typography sx={{ fontWeight: 500, fontSize: '0.875rem', display: 'inline' }}>
              {task.category.name}
            </Typography>
          </Grid>
          <Grid size={{ xs: 12, sm: 6 }}>
            <Typography sx={{ color: '#6b7280', fontSize: '0.875rem', display: 'inline' }}>
              Срок:{' '}
            </Typography>
            <Typography sx={{ fontWeight: 500, fontSize: '0.875rem', display: 'inline' }}>
              {task.dueDate || 'Не указан'}
            </Typography>
          </Grid>
          <Grid size={12}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography sx={{ color: '#6b7280', fontSize: '0.875rem' }}>Статус задачи:</Typography>
              <FormControl size='small'>
                <Select
                  value={task.status}
                  onChange={e => onStatusChange(task.id, e.target.value as TicketStatus)}
                  sx={{
                    borderRadius: '999px',
                    fontSize: '0.75rem',
                    '& .MuiOutlinedInput-notchedOutline': { borderColor: '#d1d5db' },
                  }}
                >
                  {Object.entries(STATUS_MAP).map(([value, info]) => (
                    <MenuItem key={value} value={value}>
                      {info.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Box>
          </Grid>
        </Grid>

        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1.5 }}>
            <Typography sx={{ fontWeight: 600, color: '#374151', fontSize: '0.9375rem' }}>
              Подзадачи
            </Typography>
            <Typography sx={{ fontSize: '0.75rem', color: '#6b7280' }}>
              {subtaskProgress.done}/{subtaskProgress.total}
            </Typography>
          </Box>

          {!task.subtasks || task.subtasks.length === 0 ? (
            <Typography sx={{ color: '#9ca3af', fontSize: '0.875rem' }}>Нет подзадач</Typography>
          ) : (
            <Box sx={{ display: 'flex', flexDirection: 'column' }}>
              {task.subtasks.map(sub => (
                <Box
                  key={sub.id}
                  sx={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    py: 1.25,
                    borderBottom: '1px solid #f3f4f6',
                    '&:last-child': { borderBottom: 'none' },
                  }}
                >
                  <Typography sx={{ fontSize: '0.875rem', color: '#374151', flex: 1 }}>
                    {sub.title}
                  </Typography>
                  <FormControl size='small'>
                    <Select
                      value={sub.status}
                      onChange={e =>
                        onSubtaskStatusChange(task.id, sub.id, e.target.value as TicketStatus)
                      }
                      sx={{
                        borderRadius: '999px',
                        fontSize: '0.75rem',
                        '& .MuiOutlinedInput-notchedOutline': { borderColor: '#d1d5db' },
                      }}
                    >
                      {SUBTASK_STATUS_OPTIONS.map(opt => (
                        <MenuItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Box>
              ))}
            </Box>
          )}
        </Box>
      </DialogContent>

      <DialogActions sx={{ px: 3, py: 2, borderTop: '1px solid #f3f4f6' }}>
        <Button
          onClick={onClose}
          variant='contained'
          sx={{ borderRadius: '8px', textTransform: 'none', bgcolor: '#e5e7eb', color: '#374151', ':hover': { bgcolor: '#d1d5db' } }}
        >
          Закрыть
        </Button>
      </DialogActions>
    </Dialog>
  )
}
