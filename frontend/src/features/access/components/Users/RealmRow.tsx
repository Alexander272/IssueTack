import { useMemo, type FC } from 'react'
import { Box, TableCell, TableRow } from '@mui/material'
import dayjs from 'dayjs'

import type { IUserRealm } from '@/features/user/types/user'
import { stringToHSLA } from '@/utils/colors'
import { StatusBadge } from '@/features/access/components/StatusBadge'

type Props = {
	r: IUserRealm
}

const cellStyles = {
	borderColor: '#f1f5f9',
}

type BadgeProps = {
	label: string
}

const Badge: FC<BadgeProps> = ({ label }) => {
	const { bg, text } = useMemo(() => stringToHSLA(label), [label])

	return (
		<Box
			sx={{
				display: 'inline-flex',
				alignItems: 'center',
				bgcolor: bg,
				color: text,
				px: 2,
				py: 0.75,
				borderRadius: '16px',
				fontSize: '0.875rem',
				fontWeight: 500,
			}}
		>
			{label}
		</Box>
	)
}

type RealmBadgeProps = {
	label: string
}

const RealmBadge: FC<RealmBadgeProps> = ({ label }) => {
	const { dot, text } = useMemo(() => stringToHSLA(label), [label])

	return (
		<Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1 }}>
			<Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: dot, flexShrink: 0 }} />
			<Box sx={{ fontSize: '0.875rem', fontWeight: 500, color: text }}>{label}</Box>
		</Box>
	)
}

export const RealmRow: FC<Props> = ({ r }) => {
	return (
		<TableRow>
			<TableCell sx={{ ...cellStyles }}>
				<RealmBadge label={r.realm?.name || ''} />
			</TableCell>
			<TableCell sx={{ ...cellStyles }}>
				<Badge label={r.role?.name || ''} />
			</TableCell>
			<TableCell sx={{ ...cellStyles }}>
				<StatusBadge active={r.isActive} label={r.isActive ? 'Активный' : 'Неактивный'} />
			</TableCell>
			<TableCell sx={{ ...cellStyles, color: 'text.secondary', fontSize: '13px' }}>
				{dayjs(r.createdAt).format('dddd, DD MMM YYYY HH:mm')}
			</TableCell>
		</TableRow>
	)
}
