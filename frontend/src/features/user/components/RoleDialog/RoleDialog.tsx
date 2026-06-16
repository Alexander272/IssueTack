import type { FC } from 'react'
import { Dialog, DialogContent } from '@mui/material'

import { UpdateRole, CreateRole } from '../RoleForm'

type Props = {
	roleId?: string | null
	onClose: () => void
}

export const RoleDialog: FC<Props> = ({ roleId, onClose }) => {
	return (
		<Dialog
			open={roleId != null}
			onClose={onClose}
			fullWidth
			maxWidth='xl'
			slotProps={{
				paper: {
					sx: {
						borderRadius: '16px',
						background: '#fafafa',
					},
				},
			}}
		>
			<DialogContent>
				{roleId && roleId != '' ? (
					<UpdateRole roleId={roleId} onCancel={onClose} onSuccess={onClose} />
				) : (
					<CreateRole onCancel={onClose} onSuccess={onClose} />
				)}
			</DialogContent>
		</Dialog>
	)
}
