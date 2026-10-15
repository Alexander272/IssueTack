import { type FC, useState } from 'react'
import { Button, ListItemIcon, Menu, MenuItem, Stack } from '@mui/material'
import { useNavigate } from 'react-router'
import { ArrowRightIcon, ChevronDownIcon, FolderOpenIcon, LinkIcon, PlusIcon } from 'lucide-mui'

import type { IEquipment } from '../../types/equipment'
import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { softButtonSx } from './consts'
import { useCheckPermissions } from '@/features/access/hooks/checkPerms'
import { useGetTasksQuery } from '@/features/tasks/tasksApiSlice'
import { Loader } from '@/components/Fallback/Loader'
import { TaskCreateModal } from '@/features/tasks/components/TaskCreateModal'
import { AttachTaskDialog } from './AttachTaskDialog'
import { DetailCard } from './DetailCard'
import { EmptyState } from './EmptyState'
import { EquipmentTaskItem } from './EquipmentTaskItem'

type Props = {
	unit: IEquipment
}

// Сколько заявок показываем в секции до кнопки «Показать все». Полный список —
// страница задач с фильтром по оборудованию: у одной единицы заявок может быть
// много, а секция вкладки «Обзор» не должна превращаться в ещё одну таблицу.
const PREVIEW_COUNT = 5

// Блок «Заявки по оборудованию» вместе с кнопкой создания: заявку можно завести
// новую (с уже подставленным оборудованием) или привязать существующую. Оба
// диалога — собственный контекст блока, наружу отдаётся только единица.
export const EquipmentTasks: FC<Props> = ({ unit }) => {
	const navigate = useNavigate()
	const canWrite = useCheckPermissions({ anyOf: [PermRules.Tasks.Write] })

	const [menuAnchor, setMenuAnchor] = useState<HTMLElement | null>(null)
	const [createOpen, setCreateOpen] = useState(false)
	const [attachOpen, setAttachOpen] = useState(false)

	const { data, isLoading, isError } = useGetTasksQuery({ equipmentId: unit.id })
	const tasks = data?.data ?? []
	const total = data?.total ?? tasks.length
	const preview = tasks.slice(0, PREVIEW_COUNT)

	const closeMenu = () => setMenuAnchor(null)

	// «Показать все» ведёт на страницу заявок с фильтром по этой единице: список
	// отфильтрован на месте, а чип фильтра можно снять, не теряя остальных условий.
	// Название кладём рядом с id — из него страница соберёт подпись чипа.
	const showAll = () => {
		const params = new URLSearchParams({ equipmentId: unit.id, equipmentName: unit.name })
		navigate(`${AppRoutes.Home}?${params.toString()}`)
	}

	const action = canWrite ? (
		<>
			<Button
				size='small'
				variant='contained'
				disableElevation
				startIcon={<PlusIcon sx={{ fontSize: 14 }} />}
				endIcon={<ChevronDownIcon sx={{ fontSize: 14 }} />}
				onClick={e => setMenuAnchor(e.currentTarget)}
				sx={softButtonSx}
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
		<DetailCard title={total > 0 ? `Заявки по оборудованию (${total})` : 'Заявки по оборудованию'} action={action}>
			{isLoading ? (
				<Loader />
			) : isError || tasks.length === 0 ? (
				<EmptyState
					icon={FolderOpenIcon}
					title='По этому оборудованию заявок нет'
					hint='Созданные инциденты или обслуживания появятся здесь'
				/>
			) : (
				<Stack spacing={1}>
					{preview.map(task => (
						<EquipmentTaskItem
							key={task.id}
							task={task}
							onClick={t => navigate(`${AppRoutes.Tasks}/${t.id}`)}
						/>
					))}

					{total > PREVIEW_COUNT && (
						<Button
							size='small'
							fullWidth
							endIcon={<ArrowRightIcon sx={{ fontSize: 14 }} />}
							onClick={showAll}
							sx={{
								mt: 0.5,
								borderRadius: '12px',
								textTransform: 'none',
								fontSize: 12,
								fontWeight: 600,
								color: '#2563eb',
								bgcolor: '#f8fafc',
								'&:hover': { bgcolor: '#eff6ff' },
							}}
						>
							Показать все ({total})
						</Button>
					)}
				</Stack>
			)}

			{canWrite && (
				<TaskCreateModal open={createOpen} onClose={() => setCreateOpen(false)} defaultEquipmentId={unit.id} />
			)}
			{attachOpen && <AttachTaskDialog open onClose={() => setAttachOpen(false)} unit={unit} />}
		</DetailCard>
	)
}
