import { type FC, useState } from 'react'
import { Button, ListItemIcon, Menu, MenuItem } from '@mui/material'
import { useNavigate } from 'react-router'
import { ChevronDownIcon, FolderOpenIcon, LinkIcon, PlusIcon } from 'lucide-mui'

import { Loader } from '@/components/Fallback/Loader'
import { PermRules } from '@/features/access/constants/permissions'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { TaskCreateModal } from '@/features/tasks/components/TaskCreateModal'
import { TaskCardList } from '@/features/tasks/components/Table/TaskCardList'
import { useGetTasksQuery } from '@/features/tasks/tasksApiSlice'
import { AppRoutes } from '@/pages/router/routes'
import type { IEquipment } from '../../types/equipment'
import { AttachTaskDialog } from './AttachTaskDialog'
import { DetailCard } from './DetailCard'
import { EmptyState } from './EmptyState'

type Props = {
	unit: IEquipment
}

// Блок «Заявки по оборудованию» вместе с кнопкой создания: заявку можно завести
// новую (с уже подставленным оборудованием) или привязать существующую. Оба
// диалога — собственный контекст блока, наружу отдаётся только единица.
export const EquipmentTasks: FC<Props> = ({ unit }) => {
	const navigate = useNavigate()
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Tasks.Write] })

	const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null)
	const [createOpen, setCreateOpen] = useState(false)
	const [attachOpen, setAttachOpen] = useState(false)

	const { data, isLoading, isError } = useGetTasksQuery({ equipmentId: unit.id, limit: 50 })

	const closeMenu = () => setMenuAnchor(null)

	const action = canWrite ? (
		<>
			<Button
				size='small'
				variant='contained'
				disableElevation
				startIcon={<PlusIcon sx={{ fontSize: 14 }} />}
				endIcon={<ChevronDownIcon sx={{ fontSize: 14 }} />}
				onClick={e => setMenuAnchor(e.currentTarget)}
				sx={{
					borderRadius: '10px',
					textTransform: 'none',
					fontSize: 12,
					fontWeight: 600,
					px: 1.5,
					py: 0.5,
					minWidth: 0,
				}}
			>
				Создать заявку
			</Button>
			<Menu
				anchorEl={menuAnchor}
				open={Boolean(menuAnchor)}
				onClose={closeMenu}
				slotProps={{ paper: { sx: { borderRadius: '12px', minWidth: 240, mt: 0.5 } } }}
			>
				<MenuItem
					onClick={() => {
						closeMenu()
						setCreateOpen(true)
					}}
					sx={{ fontSize: 14 }}
				>
					<ListItemIcon>
						<PlusIcon sx={{ fontSize: 16, color: '#2563eb' }} />
					</ListItemIcon>
					Создать новую заявку
				</MenuItem>
				<MenuItem
					onClick={() => {
						closeMenu()
						setAttachOpen(true)
					}}
					sx={{ fontSize: 14 }}
				>
					<ListItemIcon>
						<LinkIcon sx={{ fontSize: 16, color: '#2563eb' }} />
					</ListItemIcon>
					Привязать существующую
				</MenuItem>
			</Menu>
		</>
	) : undefined

	return (
		<DetailCard title='Заявки по оборудованию' action={action}>
			{isLoading ? (
				<Loader />
			) : isError || !data?.data || data.data.length === 0 ? (
				<EmptyState
					icon={FolderOpenIcon}
					title='По этому оборудованию заявок нет'
					hint='Созданные инциденты или обслуживания появятся здесь'
				/>
			) : (
				<TaskCardList
					tasks={data.data}
					groupBy='category'
					groupEnabled={false}
					onTaskClick={task => navigate(`${AppRoutes.Tasks}/${task.id}`)}
				/>
			)}

			{canWrite && (
				<TaskCreateModal open={createOpen} onClose={() => setCreateOpen(false)} defaultEquipmentId={unit.id} />
			)}
			{attachOpen && <AttachTaskDialog open onClose={() => setAttachOpen(false)} unit={unit} />}
		</DetailCard>
	)
}
