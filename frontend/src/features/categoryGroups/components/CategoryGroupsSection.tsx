import { useState, type FC } from 'react'
import { Box, Button, IconButton, Paper, Stack, Tooltip, Typography } from '@mui/material'
import { ChevronDownIcon, PlusIcon } from 'lucide-mui'

import type { ICategory } from '@/features/categories/types/category'
import type { ICategoryGroupDTO } from '../types/categoryGroup'
import { useGetAllCategoryGroupsQuery } from '../categoryGroupsApiSlice'
import { CategoryGroupDialog } from './Dialogs/CategoryGroupDialog'
import { CategoryOrderEditor } from '@/features/categories/components/CategoryOrderEditor'

type Props = {
	categories: ICategory[]
}

// Состояние сворачивания помнится: по умолчанию блок свёрнут (порядок
// категорий задают редко, а страница категорий должна открываться короткой),
// но если админ его развернул — в следующий раз он останется развёрнутым.
const STORAGE_KEY = 'categorySectionsCollapsed'

export const CategoryGroupsSection: FC<Props> = ({ categories }) => {
	const { data: categoryGroups } = useGetAllCategoryGroupsQuery()

	const [open, setOpen] = useState(false)
	const [editing, setEditing] = useState<ICategoryGroupDTO | null>(null)
	const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) !== 'false')

	const handleToggle = () => {
		setCollapsed(prev => {
			const next = !prev
			localStorage.setItem(STORAGE_KEY, String(next))
			return next
		})
	}

	const openCreate = () => {
		setEditing({ id: null, name: '', description: '', isHidden: false })
		setOpen(true)
	}

	const openEdit = (id: string) => {
		const cg = categoryGroups?.data?.find(g => g.id === id)
		if (!cg) return
		setEditing({
			id: cg.id,
			name: cg.name,
			description: cg.description,
			isHidden: cg.isHidden,
		})
		setOpen(true)
	}

	return (
		<Paper elevation={0} sx={{ borderRadius: 3, border: '1px solid', borderColor: 'divider', p: 2, mb: 3 }}>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'space-between',
					alignItems: 'center',
					mb: collapsed ? 0 : 2,
					gap: 2,
				}}
			>
				<Stack direction='row' spacing={1} sx={{ alignItems: 'center' }}>
					<Tooltip title={collapsed ? 'Развернуть порядок категорий' : 'Свернуть'}>
						<IconButton
							onClick={handleToggle}
							size='small'
							aria-expanded={!collapsed}
							aria-label={collapsed ? 'Развернуть блок разделов' : 'Свернуть блок разделов'}
						>
							<ChevronDownIcon
								sx={{
									fontSize: 20,
									color: '#6b7280',
									transition: 'transform 0.2s',
									transform: collapsed ? 'rotate(0deg)' : 'rotate(180deg)',
								}}
							/>
						</IconButton>
					</Tooltip>

					<Box>
						<Typography sx={{ fontWeight: 700, color: '#111827' }}>Разделы категорий</Typography>
						<Typography sx={{ fontSize: '0.8rem', color: '#6b7280' }}>
							{collapsed
								? 'Задают порядок категорий в списках выбора.'
								: 'Задают порядок категорий в списках выбора. На маршрутизацию и права не влияют.'}
						</Typography>
					</Box>
				</Stack>

				<Button
					variant='outlined'
					onClick={openCreate}
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff', whiteSpace: 'nowrap' }}
				>
					<PlusIcon sx={{ color: 'primary.main', fontSize: 16, mr: 1.5 }} />
					Создать раздел
				</Button>
			</Box>

			{!collapsed && (
				<CategoryOrderEditor
					categories={categories}
					categoryGroups={categoryGroups?.data || []}
					onEditSection={openEdit}
				/>
			)}

			<CategoryGroupDialog categoryGroup={editing || undefined} open={open} onClose={() => setOpen(false)} />
		</Paper>
	)
}
