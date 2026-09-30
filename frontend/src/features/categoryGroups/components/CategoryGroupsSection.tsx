import { useState, type FC } from 'react'
import { Box, Button, Chip, IconButton, Paper, Stack, Tooltip, Typography } from '@mui/material'
import { EditIcon, LayersIcon, PlusIcon } from 'lucide-mui'

import type { ICategory } from '@/features/categories/types/category'
import type { ICategoryGroup, ICategoryGroupDTO } from '../types/categoryGroup'
import { useGetAllCategoryGroupsQuery } from '../categoryGroupsApiSlice'
import { CategoryGroupDialog } from './Dialogs/CategoryGroupDialog'

type Props = {
	categories: ICategory[]
}

const NO_SECTION = 'Без раздела'

type GroupedSection = {
	id: string | null
	name: string
	categories: ICategory[]
}

// Порядок разделов: сначала заданные sort_order, потом категории без раздела — в конце.
const groupCategories = (categoryGroups: ICategoryGroup[], categories: ICategory[]): GroupedSection[] => {
	const sections: GroupedSection[] = categoryGroups.map(cg => ({ id: cg.id, name: cg.name, categories: [] }))
	const withoutSection: GroupedSection = { id: null, name: NO_SECTION, categories: [] }

	categories.forEach(cat => {
		const section = sections.find(s => s.id === cat.categoryGroupId) ?? withoutSection
		section.categories.push(cat)
	})

	const result = sections.filter(s => s.categories.length > 0)
	if (withoutSection.categories.length > 0) result.push(withoutSection)
	return result
}

export const CategoryGroupsSection: FC<Props> = ({ categories }) => {
	const { data: categoryGroups } = useGetAllCategoryGroupsQuery()

	const [open, setOpen] = useState(false)
	const [editing, setEditing] = useState<ICategoryGroupDTO | null>(null)

	const openCreate = () => {
		const nextOrder = (categoryGroups?.data.length ?? 0) + 1
		setEditing({ id: null, name: '', description: '', sortOrder: nextOrder })
		setOpen(true)
	}

	const openEdit = (id: string) => {
		const cg = categoryGroups?.data.find(g => g.id === id)
		if (!cg) return
		setEditing({
			id: cg.id,
			name: cg.name,
			description: cg.description,
			sortOrder: cg.sortOrder,
		})
		setOpen(true)
	}

	const sections = groupCategories(categoryGroups?.data || [], categories)

	return (
		<Paper elevation={0} sx={{ borderRadius: 3, border: '1px solid', borderColor: 'divider', p: 2, mb: 3 }}>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2, gap: 2 }}>
				<Box>
					<Typography sx={{ fontWeight: 700, color: '#111827' }}>Разделы категорий</Typography>
					<Typography sx={{ fontSize: '0.8rem', color: '#6b7280' }}>
						Группируют категории в списках выбора. На маршрутизацию и права не влияют.
					</Typography>
				</Box>
				<Button
					variant='outlined'
					onClick={openCreate}
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff', whiteSpace: 'nowrap' }}
				>
					<PlusIcon sx={{ color: 'primary.main', fontSize: 16, mr: 1.5 }} />
					Создать раздел
				</Button>
			</Box>

			<Stack spacing={1}>
				{categoryGroups?.data.map(cg => {
					const section = sections.find(s => s.id === cg.id)
					const cats = section?.categories ?? []
					return (
						<Box
							key={cg.id}
							sx={{
								display: 'flex',
								flexWrap: 'wrap',
								alignItems: 'center',
								gap: 1,
								py: 1,
								px: 1.5,
								borderRadius: '8px',
								bgcolor: '#f9fafb',
							}}
						>
							<LayersIcon sx={{ fontSize: 16, color: '#6b7280' }} />
							<Typography sx={{ fontWeight: 600, minWidth: 160 }}>{cg.name}</Typography>
							{cats.length ? (
								cats.map(cat => (
									<Chip
										key={cat.id}
										size='small'
										label={cat.name}
										sx={{ height: 22, fontSize: '0.75rem', bgcolor: '#eff6ff', color: '#1d4ed8' }}
									/>
								))
							) : (
								<Typography sx={{ fontSize: '0.8rem', color: '#9ca3af' }}>Пока нет категорий</Typography>
							)}
							<Tooltip title='Переименовать раздел'>
								<IconButton size='small' onClick={() => openEdit(cg.id)} sx={{ ml: 'auto' }}>
									<EditIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
								</IconButton>
							</Tooltip>
						</Box>
					)
				})}

				{sections
					.filter(s => s.id === null)
					.map(section => (
						<Box
							key='none'
							sx={{
								display: 'flex',
								flexWrap: 'wrap',
								alignItems: 'center',
								gap: 1,
								py: 1,
								px: 1.5,
								borderRadius: '8px',
								bgcolor: '#f9fafb',
							}}
						>
							<LayersIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
							<Typography sx={{ fontWeight: 600, color: '#6b7280', minWidth: 160 }}>{NO_SECTION}</Typography>
							{section.categories.map(cat => (
								<Chip
									key={cat.id}
									size='small'
									label={cat.name}
									sx={{ height: 22, fontSize: '0.75rem', bgcolor: '#f3f4f6', color: '#4b5563' }}
								/>
							))}
						</Box>
					))}

				{!categoryGroups?.data.length ? (
					<Typography sx={{ py: 1, color: 'text.secondary' }}>
						Разделов пока нет — все категории показываются одним списком.
					</Typography>
				) : null}
			</Stack>

			<CategoryGroupDialog categoryGroup={editing || undefined} open={open} onClose={() => setOpen(false)} />
		</Paper>
	)
}
