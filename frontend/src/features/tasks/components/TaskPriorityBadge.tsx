import { Box, Typography } from '@mui/material'
import type { Priority } from '../types/task'
import { PRIORITY_MAP } from '../constants/taskMaps'

interface Props {
  priority: Priority
}

export const TaskPriorityBadge = ({ priority }: Props) => {
  const info = PRIORITY_MAP[priority]

  return (
    <Box
      sx={{
        display: 'inline-flex',
        px: 1.25,
        py: 0.25,
        borderRadius: '999px',
        fontSize: '0.75rem',
        fontWeight: 500,
        bgcolor: info.bgColor,
        color: info.textColor,
      }}
    >
      <Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1.5 }}>
        {info.label}
      </Typography>
    </Box>
  )
}
