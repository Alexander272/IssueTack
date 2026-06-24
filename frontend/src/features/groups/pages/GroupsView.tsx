import { useCallback, useMemo, useState, type FC } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'

import type { IGroup, IGroupDTO } from '../types/group'
import { useGetAllGroupsQuery } from '../groupsApiSlice'
import { useGetAvailableUsersQuery } from '@/features/user/usersApiSlice'
import { useDebounce } from '@/hooks/useDebounce'
import { PlusIcon } from '@/components/Icons/PlusIcon'
import { GroupCard } from '../components/GroupCard'
import { GroupViewDialog } from '../components/GroupViewDialog'
import { GroupDialog } from '../components/Dialogs/GroupDialog'
import { SearchIcon } from '@/components/Icons/SearchIcon'

export const GroupsView: FC = () => {
	const { palette } = useTheme()

	const [search, setSearch] = useState('')
	const debouncedSearch = useDebounce(search, 300) as string

	const [viewGroup, setViewGroup] = useState<IGroup | null>(null)
	const [editGroup, setEditGroup] = useState<IGroupDTO | null>(null)
	const [dialogOpen, setDialogOpen] = useState(false)

	const { data: groups } = useGetAllGroupsQuery()
	const { data: usersData } = useGetAvailableUsersQuery()
	const users = usersData?.data ?? []

	const filtered = useMemo(() => {
		const q = debouncedSearch.toLowerCase()
		return groups?.data.filter(g => {
			if (!q) return true
			return (
				g.name.toLowerCase().includes(q) ||
				g.description.toLowerCase().includes(q)
			)
		})
	}, [groups, debouncedSearch])

	const openCreate = useCallback(() => {
		setEditGroup(null)
		setDialogOpen(true)
	}, [])

	const openEdit = useCallback((group: IGroup) => {
		setEditGroup({
			id: group.id,
			name: group.name,
			description: group.description,
			managerId: group.managerId ?? null,
			defaultAssigneeId: group.defaultAssigneeId ?? null,
			memberIds: group.members?.map(m => m.id) ?? [],
		})
		setDialogOpen(true)
	}, [])

	const openView = useCallback((group: IGroup) => {
		setViewGroup(group)
	}, [])

	const closeDialog = useCallback(() => {
		setEditGroup(null)
		setDialogOpen(false)
	}, [])

	return (
		<Box sx={{ p: 3 }}>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Группы исполнителей
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280' }}>
						Управление группами и их составом
					</Typography>
				</Box>
				<Button
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff' }}
					onClick={openCreate}
				>
					<PlusIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
					Создать группу
				</Button>
			</Box>

			<Box
				sx={{
					mb: 4,
					display: 'flex',
					alignItems: 'center',
					gap: 1.5,
					px: 2,
					py: 1,
					bgcolor: '#fff',
					border: '1px solid #e5e7eb',
					borderRadius: '8px',
					maxWidth: 400,
				}}
			>
				<SearchIcon sx={{ fontSize: 18, color: '#9ca3af' }} />
				<input
					value={search}
					onChange={e => setSearch(e.target.value)}
					placeholder='Поиск групп...'
					style={{
						border: 'none',
						outline: 'none',
						flex: 1,
						fontSize: '0.875rem',
						background: 'none',
						color: '#1f2937',
						fontFamily: 'inherit',
					}}
				/>
			</Box>

			<Box
				sx={{
					display: 'grid',
					gridTemplateColumns: 'repeat(auto-fill, minmax(360px, 1fr))',
					gap: 3,
				}}
			>
				{filtered?.map(group => (
					<GroupCard
						key={group.id}
						group={group}
						onView={openView}
						onEdit={openEdit}
					/>
				))}
				{filtered?.length === 0 && (
					<Box sx={{ gridColumn: '1 / -1', textAlign: 'center', py: 8 }}>
						<Typography variant='body1' sx={{ color: '#9ca3af' }}>
							{debouncedSearch ? 'Ничего не найдено' : 'Нет групп'}
						</Typography>
					</Box>
				)}
			</Box>

			<GroupViewDialog
				group={viewGroup}
				onClose={() => setViewGroup(null)}
				onEdit={group => {
					setEditGroup(group)
					setDialogOpen(true)
				}}
			/>

			<GroupDialog
				group={editGroup || undefined}
				users={users}
				open={dialogOpen}
				onClose={closeDialog}
			/>
		</Box>
	)
}
