import { Box, Typography } from '@mui/material'

interface Props {
  done: number
  total: number
}

export const TaskProgressBar = ({ done, total }: Props) => {
  const percent = total === 0 ? 0 : Math.round((done / total) * 100)

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, justifyContent: 'flex-end' }}>
      <Typography sx={{ fontSize: '0.75rem', color: '#6b7280', whiteSpace: 'nowrap' }}>
        {done}/{total}
      </Typography>
      <Box
        sx={{
          width: 64,
          height: 6,
          bgcolor: '#e2e8f0',
          borderRadius: '999px',
          overflow: 'hidden',
          flexShrink: 0,
        }}
      >
        <Box
          sx={{
            width: `${percent}%`,
            height: '100%',
            bgcolor: '#10b981',
            borderRadius: '999px',
            transition: 'width 0.2s',
          }}
        />
      </Box>
    </Box>
  )
}
