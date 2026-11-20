import { useMemo, useState, type FC } from 'react'
import { Box, Button, Chip, IconButton, Paper, Stack, Tooltip, Typography } from '@mui/material'
import { ChevronDownIcon, PencilIcon, PlusIcon } from 'lucide-mui'

import type { ICategory, ICategoryModifierDTO } from '../types/category'
import { useGetAllCategoryModifiersQuery } from '../categoriesApiSlice'
import { CategoryModifierDialog } from './Dialogs/CategoryModifierDialog'
import { PRIORITY_MAP } from '@/features/tasks/constants/taskMaps'

type Props = {
	categories: ICategory[]
}

const STORAGE_KEY = 'categoryModifiersCollapsed'

// Подкатегории — параметры разовой подстановки срока/приоритета. Блок свёрнут по
// умолчанию: страница категорий и без него длинная, а настраивают подкатегории редко.
export const CategoryModifiersSection: FC<Props> = ({ categories }) => {
	const { data: modifiersData } = useGetAllCategoryModifiersQuery()
	const [collapsed, setCollapsed] = useState(() => localStorage.getItem(STORAGE_KEY) !== 'false')
	const [editing, setEditing] = useState<ICategoryModifierDTO | null>(null)
	const [dialogOpen, setDialogOpen] = useState(false)

	const categoryNames = useMemo(() => new Map(categories.map(c => [c.id, c.name])), [categories])
	const modifiers = useMemo(() => modifiersData?.data ?? [], [modifiersData])

	const toggle = () =>
		setCollapsed(prev => {
			const next = !prev
			localStorage.setItem(STORAGE_KEY, String(next))
			return next
		})

	const openCreate = () => {
		setEditing({ id: null, categoryId: categories[0]?.id ?? '', name: '', description: '', defaultDueDays: null, priority: null, isActive: true })
		setDialogOpen(true)
	}

	const openEdit = (id: string) => {
		const m = modifiers.find(item => item.id === id)
		if (!m) return
		setEditing({
			id: m.id,
			categoryId: m.categoryId,
			name: m.name,
			description: m.description,
			defaultDueDays: m.defaultDueDays,
			priority: m.priority,
			isActive: m.isActive,
		})
		setDialogOpen(true)
	}

	return (
		<Paper elevation={0} sx={{ borderRadius: 3, border: '1px solid', borderColor: 'divider', p: 2, mb: 3 }}>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 2, mb: collapsed ? 0 : 2 }}>
				<Stack direction='row' spacing={1} sx={{ alignItems: 'center' }}>
					<Tooltip title={collapsed ? 'Развернуть подкатегории' : 'Свернуть'}>
						<IconButton onClick={toggle} size='small' aria-expanded={!collapsed} aria-label='Свернуть блок подкатегорий'>
							<ChevronDownIcon
								sx={{ fontSize: 20, color: '#6b7280', transition: 'transform 0.2s', transform: collapsed ? 'rotate(0deg)' : 'rotate(180deg)' }}
							/>
						</IconButton>
					</Tooltip>
					<Box>
						<Typography sx={{ fontWeight: 700, color: '#111827' }}>Подкатегории</Typography>
						<Typography sx={{ fontSize: '0.8rem', color: '#6b7280' }}>
							Свой срок и приоритет. Применяются один раз при выборе в новой заявке.
						</Typography>
					</Box>
				</Stack>
				<Button
					variant='outlined'
					onClick={openCreate}
					disabled={categories.length === 0}
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff', whiteSpace: 'nowrap' }}
				>
					<PlusIcon sx={{ color: 'primary.main', fontSize: 16, mr: 1.5 }} />
					Создать подкатегорию
				</Button>
			</Box>

			{!collapsed && (
				<Stack spacing={1}>
					{modifiers.length === 0 && (
						<Typography variant='body2' sx={{ color: 'text.secondary' }}>
							Подкатегорий пока нет.
						</Typography>
					)}
					{modifiers.map(m => (
						<Box
							key={m.id}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1.5,
								p: 1,
								px: 1.5,
								border: '1px solid',
								borderColor: 'divider',
								borderRadius: 2,
								opacity: m.isActive ? 1 : 0.6,
							}}
						>
							<Box sx={{ minWidth: 0, flex: 1 }}>
								<Typography sx={{ fontSize: '0.875rem', fontWeight: 600 }}>{m.name}</Typography>
								<Typography sx={{ fontSize: '0.75rem', color: 'text.secondary' }}>
									{categoryNames.get(m.categoryId) ?? '—'}
									{m.description ? ` · ${m.description}` : ''}
								</Typography>
							</Box>
							<Chip
								size='small'
								variant='outlined'
								label={m.defaultDueDays != null ? `${m.defaultDueDays} дн.` : 'Срок категории'}
							/>
							<Chip
								size='small'
								variant='outlined'
								label={m.priority ? PRIORITY_MAP[m.priority].label : 'Приоритет категории'}
								sx={{ display: { xs: 'none', sm: 'inline-flex' } }}
							/>
							<IconButton size='small' onClick={() => openEdit(m.id)} aria-label='Редактировать подкатегорию'>
								<PencilIcon sx={{ fontSize: 16 }} />
							</IconButton>
						</Box>
					))}
				</Stack>
			)}

			<CategoryModifierDialog
				modifier={editing || undefined}
				categories={categories}
				open={dialogOpen}
				onClose={() => setDialogOpen(false)}
			/>
		</Paper>
	)
}
