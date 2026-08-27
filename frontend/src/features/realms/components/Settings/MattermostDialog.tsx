import { useState, type FC } from 'react'
import { Dialog, DialogTitle, DialogContent, IconButton, Stack, Tab, Tabs, Typography } from '@mui/material'
import { XIcon, MessageSquareIcon } from 'lucide-mui'

import { MattermostTab } from './MattermostTab'

type Props = {
	realmId: string
	open: boolean
	onClose: () => void
}

export const MattermostDialog: FC<Props> = ({ realmId, open, onClose }) => {
	const [tab, setTab] = useState('mattermost')

	return (
		<Dialog
			open={open}
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
					Интеграции
				</Typography>
				<IconButton onClick={onClose} sx={{ color: 'text.secondary' }}>
					<XIcon sx={{ fontSize: 16 }} />
				</IconButton>
			</DialogTitle>

			<DialogContent dividers sx={{ borderTop: '1px solid #f0f0f0', borderBottom: '1px solid #f0f0f0', px: 0 }}>
				<Tabs
					value={tab}
					onChange={(_, v) => setTab(v)}
					sx={{
						px: 3,
						minHeight: 36,
						'& .MuiTab-root': { minHeight: 36, py: 0, textTransform: 'none' },
					}}
				>
					<Tab
						value='mattermost'
						label='Mattermost'
						iconPosition='start'
						icon={<MessageSquareIcon sx={{ fontSize: 14 }} />}
					/>
				</Tabs>

				<Stack sx={{ px: 3, pt: 3 }}>
					{tab === 'mattermost' && <MattermostTab realmId={realmId} />}
				</Stack>
			</DialogContent>
		</Dialog>
	)
}
