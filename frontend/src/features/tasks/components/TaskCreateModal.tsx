import { type FC, useRef, useState } from 'react'
import { Box, Dialog, DialogContent, DialogTitle, IconButton, Tooltip, Typography, useMediaQuery, useTheme } from '@mui/material'

import { TaskCreateForm, type TaskCreateFormHandle } from './TaskCreateForm'
import { EraserIcon, XIcon } from 'lucide-mui'

type Props = {
	open: boolean
	onClose: () => void
}

export const TaskCreateModal: FC<Props> = ({ open, onClose }) => {
	const theme = useTheme()
	const isMobile = useMediaQuery(theme.breakpoints.down('sm'))
	const [saving, setSaving] = useState(false)
	const [dirty, setDirty] = useState(false)
	const formRef = useRef<TaskCreateFormHandle>(null)

	const canClose = !saving

	// Пока в форме есть непустой черновик, клик мимо и Escape не закрывают окно —
	// иначе заполненное теряется от случайного клика. Закрыть можно крестиком или
	// после успешного создания/очистки. Крестик вызывает onClose напрямую (не
	// через reason), поэтому остаётся доступным.
	const handleClose = (_event: unknown, reason: 'backdropClick' | 'escapeKeyDown') => {
		if (saving) return
		if (dirty && (reason === 'backdropClick' || reason === 'escapeKeyDown')) return
		onClose()
	}

	return (
		<Dialog
			open={open}
			onClose={handleClose}
			fullWidth
			fullScreen={isMobile}
			maxWidth='md'
			slotProps={{
				paper: { sx: { borderRadius: { xs: 0, sm: '16px' }, p: { xs: 0, sm: 1 } } },
			}}
		>
			<DialogTitle
				sx={{ m: 0, p: { xs: 1.5, sm: 2 }, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
			>
				<Typography variant='h6' component='div' sx={{ fontWeight: 'bold' }}>
					Создание заявки
				</Typography>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
					{dirty && (
						<Tooltip title='Очистить черновик'>
							<span>
								<IconButton
									size='large'
									onClick={() => formRef.current?.requestReset()}
									disabled={saving}
									sx={{ color: 'text.secondary' }}
								>
									<EraserIcon sx={{ fontSize: 20 }} />
								</IconButton>
							</span>
						</Tooltip>
					)}
					<IconButton size='large' onClick={canClose ? onClose : undefined} disabled={saving} sx={{ color: 'text.secondary' }}>
						<XIcon sx={{ fontSize: 20 }} />
					</IconButton>
				</Box>
			</DialogTitle>

			<DialogContent sx={{ p: { xs: 1, sm: 2.5 } }}>
				<TaskCreateForm
					ref={formRef}
					embedded
					onSuccess={onClose}
					onCancel={onClose}
					onSavingChange={setSaving}
					onDirtyChange={setDirty}
				/>
			</DialogContent>
		</Dialog>
	)
}
