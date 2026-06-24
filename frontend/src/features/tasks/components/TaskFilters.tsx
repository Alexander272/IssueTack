import {
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  Button,
  Checkbox,
  FormControlLabel,
  InputAdornment,
} from '@mui/material'

import type { TicketStatus } from '../types/task'
import {
  QUEUE_OPTIONS,
  STATUS_OPTIONS,
  SORT_OPTIONS,
  GROUP_BY_OPTIONS,
  type GroupByField,
} from '../constants/taskMaps'

export interface FilterValues {
  queue: string
  status: TicketStatus | 'all'
  sort: string
  search: string
  groupBy: GroupByField
  groupEnabled: boolean
}

interface Props {
  filters: FilterValues
  onChange: (patch: Partial<FilterValues>) => void
  onReset: () => void
}

export const TaskFilters = ({ filters, onChange, onReset }: Props) => {
  return (
    <Box
      sx={{
        bgcolor: '#fff',
        p: 2.5,
        borderRadius: '12px',
        border: '1px solid #e5e7eb',
        mb: 3,
      }}
    >
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, alignItems: 'flex-end' }}>
        <FormControl size='small' sx={{ minWidth: 160, flex: '1 1 150px' }}>
          <InputLabel sx={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Очередь
          </InputLabel>
          <Select
            value={filters.queue}
            label='Очередь'
            onChange={e => onChange({ queue: e.target.value })}
            sx={{ borderRadius: '8px' }}
          >
            {QUEUE_OPTIONS.map(opt => (
              <MenuItem key={opt.value} value={opt.value}>
                {opt.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl size='small' sx={{ minWidth: 160, flex: '1 1 150px' }}>
          <InputLabel sx={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Статус
          </InputLabel>
          <Select
            value={filters.status}
            label='Статус'
            onChange={e => onChange({ status: e.target.value as TicketStatus | 'all' })}
            sx={{ borderRadius: '8px' }}
          >
            {STATUS_OPTIONS.map(opt => (
              <MenuItem key={opt.value} value={opt.value}>
                {opt.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl size='small' sx={{ minWidth: 180, flex: '1 1 180px' }}>
          <InputLabel sx={{ fontSize: '0.75rem', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
            Сортировка
          </InputLabel>
          <Select
            value={filters.sort}
            label='Сортировка'
            onChange={e => onChange({ sort: e.target.value })}
            sx={{ borderRadius: '8px' }}
          >
            {SORT_OPTIONS.map(opt => (
              <MenuItem key={opt.value} value={opt.value}>
                {opt.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <Box sx={{ display: 'flex', alignItems: 'flex-end', gap: 1.5 }}>
          <FormControlLabel
            control={
              <Checkbox
                checked={filters.groupEnabled}
                onChange={e => onChange({ groupEnabled: e.target.checked })}
                size='small'
                sx={{ '&.Mui-checked': { color: 'primary.main' } }}
              />
            }
            label='Группировать'
            sx={{ '& .MuiTypography-root': { fontSize: '0.875rem', fontWeight: 500 } }}
          />
          <FormControl size='small' sx={{ minWidth: 160 }}>
            <Select
              value={filters.groupBy}
              onChange={e => onChange({ groupBy: e.target.value as GroupByField })}
              sx={{ borderRadius: '8px' }}
              disabled={!filters.groupEnabled}
            >
              {GROUP_BY_OPTIONS.map(opt => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Box>

        <Button
          variant='text'
          size='small'
          onClick={onReset}
          sx={{ textTransform: 'none', color: '#6b7280', fontWeight: 500, minWidth: 'auto' }}
        >
          Сбросить
        </Button>
      </Box>

      <TextField
        size='small'
        placeholder='Поиск...'
        value={filters.search}
        onChange={e => onChange({ search: e.target.value })}
        sx={{
          mt: 2,
          width: '100%',
          maxWidth: 400,
          '& .MuiOutlinedInput-root': { borderRadius: '8px' },
        }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position='start'>
                <Box component='span' sx={{ color: '#9ca3af', fontSize: '0.875rem' }}>🔍</Box>
              </InputAdornment>
            ),
          },
        }}
      />
    </Box>
  )
}
