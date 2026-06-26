import { useMemo, useState, type FC } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'

import type { ICategory, ICategoryDTO } from '../types/category'
import { useGetAllCategoriesQuery } from '../categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useDebounce } from '@/hooks/useDebounce'
import { PlusIcon } from '@/components/Icons/PlusIcon'
import { CategoryFilters, type CategoryFilterState } from '../components/CategoryFilters'
import { CategoryCardList } from '../components/CategoryCardList'
import { CategoryTable } from '../components/CategoryTable'
import { CategoryViewDialog } from '../components/CategoryViewDialog'
import { CategoryDialog } from '../components/Dialogs/CategoryDialog'

export const CategoriesView: FC = () => {
	const { palette } = useTheme()

	const [open, setOpen] = useState(false)
	const [category, setCategory] = useState<ICategoryDTO | null>(null)
	const [viewCategory, setViewCategory] = useState<ICategory | null>(null)

	const [filters, setFilters] = useState<CategoryFilterState>({
		group: 'all',
		status: 'all',
		priority: 'all',
		search: '',
	})
	const debouncedSearch = useDebounce(filters.search, 300) as string

	const { data: categories } = useGetAllCategoriesQuery()
	const { data: groups } = useGetAllGroupsQuery()

	const groupsMap = useMemo(() => {
		const map = new Map<string, string>()
		groups?.data.forEach(g => map.set(g.id, g.name))
		return map
	}, [groups?.data])

	const filtered = useMemo(() => {
		const q = debouncedSearch.toLowerCase()
		return categories?.data.filter(c => {
			if (filters.group !== 'all' && c.groupId !== filters.group) return false
			if (filters.status === 'active' && !c.isActive) return false
			if (filters.status === 'inactive' && c.isActive) return false
			if (filters.priority !== 'all' && c.priority !== filters.priority) return false
			if (q && !c.name.toLowerCase().includes(q) && !c.description.toLowerCase().includes(q)) return false
			return true
		})
	}, [categories, filters, debouncedSearch])

	const openCreate = () => {
		setCategory(null)
		setOpen(true)
	}

	const openEdit = (cat: ICategory) => {
		setCategory({
			id: cat.id,
			name: cat.name,
			description: cat.description,
			groupId: cat.groupId,
			priority: cat.priority,
			isActive: cat.isActive,
		})
		setOpen(true)
	}

	const openView = (cat: ICategory) => {
		setViewCategory(cat)
	}

	const closeDialog = () => {
		setCategory(null)
		setOpen(false)
	}

	const resetFilters = () => {
		setFilters({ group: 'all', status: 'all', priority: 'all', search: '' })
	}

	return (
		<Box sx={{ p: { xs: 2, sm: 3 } }}>
			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 0 },
					mb: 4,
				}}
			>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Категории задач
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						Управление категориями и их привязкой к группам исполнителей
					</Typography>
				</Box>
				<Button
					variant='outlined'
					sx={{ borderRadius: '8px', textTransform: 'none', background: '#fff', width: { xs: '100%', sm: 'auto' } }}
					onClick={openCreate}
				>
					<PlusIcon fill={palette.primary.main} fontSize={16} mr={1.5} />
					Создать категорию
				</Button>
			</Box>

			<CategoryFilters
				groups={groups?.data || []}
				filters={filters}
				onChange={patch => setFilters(prev => ({ ...prev, ...patch }))}
				onReset={resetFilters}
			/>

			<Box sx={{ display: { xs: 'none', md: 'block' } }}>
				<CategoryTable categories={filtered || []} groupsMap={groupsMap} onView={openView} onEdit={openEdit} />
			</Box>
			<Box sx={{ display: { xs: 'block', md: 'none' } }}>
				<CategoryCardList categories={filtered || []} groupsMap={groupsMap} onView={openView} onEdit={openEdit} />
			</Box>

			<CategoryViewDialog
				category={viewCategory}
				groupsMap={groupsMap}
				onClose={() => setViewCategory(null)}
				onEdit={cat => {
					setViewCategory(null)
					setCategory(cat)
					setOpen(true)
				}}
			/>

			<CategoryDialog
				category={category || undefined}
				groups={groups?.data || []}
				open={open}
				onClose={closeDialog}
			/>
		</Box>
	)
}
