import { type FC } from 'react'
import {
	Box,
	Paper,
	Typography,
	Button,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Avatar,
	Tooltip,
	useTheme,
} from '@mui/material'

import type { IUserData } from '@/features/user/types/user'
import { RealmRow } from './RealmRow'
import { getAvatarColor, getInitials } from './utils'
import { ModifyIcon } from '@/components/Icons/ModifyIcon'

type Props = {
	user: IUserData
	onClick: (user: IUserData | null, type: 'edit' | 'logins') => void
}

const cellStyles = {
	color: '#475569',
	fontSize: '0.75rem',
	fontWeight: 'bold',
	borderColor: '#e2e8f0',
}

export const UserCard: FC<Props> = ({ user, onClick }) => {
	const { palette } = useTheme()

	const editHandler = (e: React.MouseEvent) => {
		e.stopPropagation()
		e.preventDefault()
		onClick(user, 'edit')
	}

	// const openHandler = (e: React.MouseEvent) => {
	// 	e.stopPropagation()
	// 	e.preventDefault()
	// 	onClick(user, 'logins')
	// }

	return (
		<Paper
			sx={{
				border: '1px solid #eee',
				borderRadius: 4,
				overflow: 'hidden',
				boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
			}}
		>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
					px: 3,
					py: 1.5,
					background: '#faf9ff',
					borderBottom: '1px solid #e2e8f0',
				}}
			>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
					<Avatar
						sx={{
							bgcolor: getAvatarColor(user.id),
							fontSize: '14px',
							width: 36,
							height: 36,
						}}
					>
						{getInitials(user)}
					</Avatar>
					<Box>
						<Typography variant='body2' sx={{ fontWeight: 600 }}>
							{user.firstName && user.lastName ? `${user.firstName} ${user.lastName}` : user.username}
						</Typography>
						<Typography variant='caption' color='text.secondary' sx={{ display: 'block' }}>
							{user.email}
						</Typography>
					</Box>
				</Box>

				<Tooltip title='Редактировать пользователя'>
					<Button
						onClick={editHandler}
						sx={{
							minWidth: 40,
							minHeight: 40,
							borderRadius: '6px',
							':hover': { svg: { fill: palette.secondary.main } },
						}}
					>
						<ModifyIcon sx={{ fontSize: 18 }} />
					</Button>
				</Tooltip>
			</Box>

			{user?.realms?.length > 0 && user.realms[0].role ? (
				<TableContainer sx={{ px: 3, py: 2 }}>
					<Table size='small'>
						<TableHead>
							<TableRow>
								<TableCell
									sx={{
										...cellStyles,
										width: 200,
									}}
								>
									Область
								</TableCell>
								<TableCell sx={{ ...cellStyles, width: 250 }}>Роль</TableCell>
								<TableCell sx={{ ...cellStyles, width: 200 }}>Статус</TableCell>
								<TableCell sx={{ ...cellStyles, width: 250 }}>Создан</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{user.realms.map(r => (
								<RealmRow key={r.realmId} r={r} />
							))}
						</TableBody>
					</Table>
				</TableContainer>
			) : (
				<Box sx={{ px: 3, py: 2 }}>
					<Typography variant='body2' color='text.secondary' align='center'>
						Нет областей
					</Typography>
				</Box>
			)}
		</Paper>
	)
}
