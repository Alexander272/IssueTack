import { Box, Typography } from '@mui/material'
import type { TicketStatus } from '../types/task'
import { STATUS_MAP } from '../constants/taskMaps'

interface Props {
  status: TicketStatus
}

export const TaskStatusBadge = ({ status }: Props) => {
  const info = STATUS_MAP[status]

  return (
    <Box
      sx={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 0.75,
        px: 1.25,
        py: 0.5,
        borderRadius: '999px',
        fontSize: '0.75rem',
        fontWeight: 500,
        bgcolor: info.bgColor,
        color: info.textColor,
      }}
    >
      <Box
        component='span'
        sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: info.dotColor, flexShrink: 0 }}
      />
      <Typography component='span' sx={{ fontSize: '0.75rem', fontWeight: 500, lineHeight: 1 }}>
        {info.label}
      </Typography>
    </Box>
  )
}
