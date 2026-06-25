import { Box, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import dayjs from 'dayjs'

import type { ISite } from '../types/site'
import { BusinessIcon } from '@/components/Icons/BusinessIcon'
import { EyeIcon } from '@/components/Icons/EyeIcon'
import { EditIcon } from '@/components/Icons/EditIcon'

type Props = {
	sites: ISite[]
	onView: (site: ISite) => void
	onEdit: (site: ISite) => void
}

const labelSx = { fontSize: '0.75rem', color: '#9ca3af' }
const valueSx = { fontSize: '0.75rem', color: '#6b7280' }

export const SiteCardList = ({ sites, onView, onEdit }: Props) => {
	return (
		<Stack sx={{ gap: 2 }}>
			{sites.map(site => (
				<Box
					key={site.id}
					sx={{
						bgcolor: '#fff',
						borderRadius: '12px',
						border: '1px solid #e5e7eb',
						boxShadow: '0 1px 2px 0 rgba(0,0,0,0.05)',
						overflow: 'hidden',
					}}
				>
					<Box sx={{ p: 2 }}>
						<Box sx={{ display: 'flex', gap: 2, mb: 2 }}>
							<Box
								sx={{
									width: 44,
									height: 44,
									borderRadius: '10px',
									bgcolor: '#dbeafe',
									fill: '#2563eb',
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
									flexShrink: 0,
								}}
							>
								<BusinessIcon sx={{ fontSize: 18 }} />
							</Box>
							<Box sx={{ flex: 1, minWidth: 0 }}>
								<Typography
									sx={{
										fontWeight: 700,
										color: '#111827',
										fontSize: '0.9375rem',
										overflow: 'hidden',
										textOverflow: 'ellipsis',
										whiteSpace: 'nowrap',
									}}
								>
									{site.name}
								</Typography>
								<Typography sx={{ fontSize: '0.8125rem', color: '#6b7280', mt: 0.25 }}>
									{site.address}
								</Typography>
							</Box>
						</Box>

						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mb: 2 }}>
							<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
								<Typography sx={labelSx}>Создана</Typography>
								<Typography sx={valueSx}>
									{dayjs(site.createdAt).locale('ru').format('DD MMMM YYYY')}
								</Typography>
							</Box>
							<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
								<Typography sx={labelSx}>Обновлена</Typography>
								<Typography sx={valueSx}>
									{dayjs(site.updatedAt).locale('ru').format('DD MMMM YYYY')}
								</Typography>
							</Box>
						</Box>

						<Box
							sx={{
								display: 'flex',
								alignItems: 'center',
								justifyContent: 'space-between',
								pt: 1.5,
								borderTop: '1px solid #f3f4f6',
							}}
						>
							<Typography sx={labelSx}>
								ID: {site.id.slice(0, 8)}...
							</Typography>
							<Box sx={{ display: 'flex', gap: 0.5 }}>
								<Tooltip title='Просмотр'>
									<IconButton
										onClick={() => onView(site)}
										size='small'
										sx={{ color: '#9ca3af', '&:hover': { color: '#3b82f6', bgcolor: '#f3f4f6' } }}
									>
										<EyeIcon sx={{ fontSize: 16 }} />
									</IconButton>
								</Tooltip>
								<Tooltip title='Редактировать'>
									<IconButton
										onClick={() => onEdit(site)}
										size='small'
										sx={{ color: '#9ca3af', '&:hover': { color: '#f59e0b', bgcolor: '#f3f4f6' } }}
									>
										<EditIcon sx={{ fontSize: 16 }} />
									</IconButton>
								</Tooltip>
							</Box>
						</Box>
					</Box>
				</Box>
			))}
		</Stack>
	)
}
