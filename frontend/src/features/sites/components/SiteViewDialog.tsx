import { type FC } from 'react'
import { Box, Button, Dialog, DialogActions, DialogContent, DialogTitle, IconButton, Typography } from '@mui/material'
import dayjs from 'dayjs'

import type { ISite, ISiteDTO } from '../types/site'
import { BusinessIcon } from '@/components/Icons/BusinessIcon'
import { TimesIcon } from '@/components/Icons/TimesIcon'

type Props = {
	site: ISite | null
	onClose: () => void
	onEdit: (site: ISiteDTO) => void
}

const fieldLabelSx = { fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }

export const SiteViewDialog: FC<Props> = ({ site, onClose, onEdit }) => {
	if (!site) return null

	return (
		<Dialog
			open={Boolean(site)}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{
				paper: { sx: { borderRadius: '16px', p: 1 } },
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Просмотр площадки
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
				<Box sx={{ display: 'flex', gap: 2, mb: 4, alignItems: 'center' }}>
					<Box
						sx={{
							width: 56,
							height: 56,
							borderRadius: '12px',
							bgcolor: '#dbeafe',
							fill: '#2563eb',
							display: 'flex',
							alignItems: 'center',
							justifyContent: 'center',
							flexShrink: 0,
						}}
					>
						<BusinessIcon sx={{ fontSize: 28 }} />
					</Box>
					<Box>
						<Typography variant='h5' sx={{ fontWeight: 'bold' }}>
							{site.name}
						</Typography>
					</Box>
				</Box>

				<Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' }, gap: 3 }}>
					<Box sx={{ gridColumn: { xs: '1', sm: '1 / -1' } }}>
						<Typography variant='caption' sx={fieldLabelSx}>
							Адрес
						</Typography>
						<Typography color='text.primary'>{site.address}</Typography>
					</Box>
					<Box>
						<Typography variant='caption' sx={fieldLabelSx}>
							ID площадки
						</Typography>
						<Typography
							component='code'
							sx={{
								fontSize: '0.75rem',
								bgcolor: '#f3f4f6',
								px: 1,
								py: 0.5,
								borderRadius: '4px',
								color: '#6b7280',
							}}
						>
							{site.id}
						</Typography>
					</Box>
				</Box>

				<Box
					sx={{
						mt: 3,
						pt: 2,
						borderTop: '1px solid #f0f0f0',
						display: 'grid',
						gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
						gap: 2,
					}}
				>
					<Typography variant='caption' color='text.secondary'>
						<strong>Создана:</strong> {dayjs(site.createdAt).locale('ru').format('DD MMMM YYYY, HH:mm')}
					</Typography>
					<Typography variant='caption' color='text.secondary'>
						<strong>Обновлена:</strong> {dayjs(site.updatedAt).locale('ru').format('DD MMMM YYYY, HH:mm')}
					</Typography>
				</Box>
			</DialogContent>

			<DialogActions sx={{ p: 2, gap: 1 }}>
				<Button
					onClick={onClose}
					variant='outlined'
					sx={{ textTransform: 'none', color: 'text.primary', borderColor: '#ddd' }}
				>
					Закрыть
				</Button>
				<Button
					onClick={() => onEdit({ id: site.id, name: site.name, address: site.address })}
					variant='contained'
					sx={{ textTransform: 'none', px: 3 }}
				>
					Редактировать
				</Button>
			</DialogActions>
		</Dialog>
	)
}
