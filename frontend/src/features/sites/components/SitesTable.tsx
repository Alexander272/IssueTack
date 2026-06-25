import { type FC } from 'react'
import {
	Box,
	IconButton,
	Paper,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Tooltip,
	Typography,
} from '@mui/material'

import type { ISite } from '../types/site'
import dayjs from 'dayjs'
import 'dayjs/locale/ru'
import { BusinessIcon } from '@/components/Icons/BusinessIcon'
import { EditIcon } from '@/components/Icons/EditIcon'
import { EyeIcon } from '@/components/Icons/EyeIcon'

type Props = {
	sites: ISite[]
	onView: (site: ISite) => void
	onEdit: (site: ISite) => void
}

export const SitesTable: FC<Props> = ({ sites, onView, onEdit }) => {
	return (
		<TableContainer
			component={Paper}
			elevation={0}
			sx={{ borderRadius: '24px', border: '1px solid #f3f4f6', overflow: 'hidden', overflowX: 'auto' }}
		>
			<Table sx={{ minWidth: 900 }}>
				<TableHead>
					<TableRow sx={{ borderBottom: '1px solid #f3f4f6' }}>
						<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
							Название
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 4, color: 'text.secondary', fontSize: '0.875rem' }}>
							Адрес
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Создана
						</TableCell>
						<TableCell sx={{ py: 2.5, px: 3, color: 'text.secondary', fontSize: '0.875rem' }}>
							Обновлена
						</TableCell>
						<TableCell align='right' sx={{ p: 0, width: 120 }}></TableCell>
					</TableRow>
				</TableHead>

				<TableBody sx={{ '& tr:not(:last-child)': { borderBottom: '1px solid #f3f4f6' } }}>
					{sites.map(site => (
						<TableRow
							key={site.id}
							hover
							sx={{
								'&:hover': { bgcolor: '#fafafa' },
							}}
						>
							<TableCell sx={{ py: 2, px: 4 }}>
								<Stack direction='row' spacing={2} sx={{ alignItems: 'center' }}>
									<Box
										sx={{
											width: 36,
											height: 36,
											borderRadius: '8px',
											bgcolor: '#dbeafe',
											fill: '#2563eb',
											display: 'flex',
											alignItems: 'center',
											justifyContent: 'center',
											flexShrink: 0,
										}}
									>
										<BusinessIcon sx={{ fontSize: 16 }} />
									</Box>
									<Typography sx={{ fontWeight: 600, color: '#111827', fontSize: '0.875rem' }}>
										{site.name}
									</Typography>
								</Stack>
							</TableCell>

							<TableCell sx={{ py: 2, px: 4 }}>
								<Typography
									sx={{
										fontSize: '0.875rem',
										color: '#6b7280',
										maxWidth: 320,
										overflow: 'hidden',
										textOverflow: 'ellipsis',
										whiteSpace: 'nowrap',
									}}
								>
									{site.address}
								</Typography>
							</TableCell>

							<TableCell sx={{ py: 2, px: 3, color: '#6b7280', fontSize: '0.8125rem' }}>
								{dayjs(site.createdAt).locale('ru').format('DD MMMM YYYY')}
							</TableCell>

							<TableCell sx={{ py: 2, px: 3, color: '#6b7280', fontSize: '0.8125rem' }}>
								{dayjs(site.updatedAt).locale('ru').format('DD MMMM YYYY')}
							</TableCell>

							<TableCell align='center' sx={{ p: 0, pr: 1 }}>
								<Stack direction='row' sx={{ justifyContent: 'flex-end' }}>
									<Tooltip title='Просмотр'>
										<IconButton
											onClick={() => onView(site)}
											size='large'
											sx={{ ':hover': { svg: { fill: '#f59e0b' } } }}
										>
											<EyeIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
										</IconButton>
									</Tooltip>
									<Tooltip title='Редактировать'>
										<IconButton
											onClick={() => onEdit(site)}
											size='large'
											sx={{ ':hover': { svg: { fill: '#3b82f6' } } }}
										>
											<EditIcon sx={{ fontSize: 18, fill: '#9ca3af' }} />
										</IconButton>
									</Tooltip>
								</Stack>
							</TableCell>
						</TableRow>
					))}
					{!sites.length ? (
						<TableRow>
							<TableCell colSpan={5} align='center' sx={{ py: 3, color: 'text.secondary' }}>
								Площадки не найдены.
							</TableCell>
						</TableRow>
					) : null}
				</TableBody>
			</Table>
		</TableContainer>
	)
}
