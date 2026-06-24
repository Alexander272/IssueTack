import { type FC } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	IconButton,
	Stack,
	Typography,
} from '@mui/material'

import type { ICategory, ICategoryDTO } from '../types/category'
import { getSmartDate } from '@/utils/date'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'
import { StatusBadge } from '@/features/access/components/StatusBadge'
import { TimesIcon } from '@/components/Icons/TimesIcon'

type Props = {
	category: ICategory | null
	groupsMap: Map<string, string>
	onClose: () => void
	onEdit: (cat: ICategoryDTO) => void
}

export const CategoryViewDialog: FC<Props> = ({ category, groupsMap, onClose, onEdit }) => {
	return (
		<Dialog
			open={Boolean(category)}
			onClose={onClose}
			fullWidth
			maxWidth='sm'
			slotProps={{
				paper: {
					sx: { borderRadius: '16px', p: 1 },
				},
			}}
		>
			<DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Просмотр категории
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<TimesIcon fontSize={16} />
				</IconButton>
			</DialogTitle>
			{category && (
				<>
					<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', py: 3 }}>
						<Stack spacing={3}>
							<Box>
								<Typography variant='h5' sx={{ fontWeight: 'bold' }}>
									{category.name}
								</Typography>
								<Box sx={{ mt: 1 }}>
									<StatusBadge
										active={category.isActive}
										label={category.isActive ? 'Активна' : 'Неактивна'}
									/>
								</Box>
							</Box>

							<Box>
								<Typography variant='caption' sx={{ fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }}>
									Описание
								</Typography>
								<Typography color='text.primary'>
									{category.description || '—'}
								</Typography>
							</Box>

							<Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 3 }}>
								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }}>
										Группа-владелец
									</Typography>
									<Typography color='text.primary'>
										{groupsMap.get(category.groupId) || 'Неизвестная группа'}
									</Typography>
								</Box>
								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }}>
										Приоритет по умолчанию
									</Typography>
									<Typography color='text.primary'>
										{PRIORITY_MAP[category.priority]?.label || category.priority}
									</Typography>
								</Box>
								<Box>
									<Typography variant='caption' sx={{ fontWeight: 600, display: 'block', color: 'text.secondary', mb: 0.5 }}>
										ID категории
									</Typography>
									<Typography
										component='code'
										sx={{ fontSize: '0.75rem', bgcolor: '#f3f4f6', px: 1, py: 0.5, borderRadius: '4px', color: '#6b7280' }}
									>
										{category.id}
									</Typography>
								</Box>
							</Box>

							<Box sx={{ pt: 2, borderTop: '1px solid #f0f0f0', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>
								<Typography variant='caption' color='text.secondary'>
									<strong>Создана:</strong> {getSmartDate(category.createdAt)}
								</Typography>
								<Typography variant='caption' color='text.secondary'>
									<strong>Обновлена:</strong> {getSmartDate(category.updatedAt)}
								</Typography>
							</Box>
						</Stack>
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
							onClick={() => {
								onEdit({
									id: category.id,
									name: category.name,
									description: category.description,
									groupId: category.groupId,
									priority: category.priority,
									isActive: category.isActive,
								})
							}}
							variant='contained'
							sx={{ textTransform: 'none', px: 3 }}
						>
							Редактировать
						</Button>
					</DialogActions>
				</>
			)}
		</Dialog>
	)
}
