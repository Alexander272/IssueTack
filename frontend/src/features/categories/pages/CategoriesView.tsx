import { useState, type FC } from 'react'
import { Button, MenuItem, MenuList, Popover, Box, Typography } from '@mui/material'
import { CornerDownRightIcon, FolderIcon, PlusIcon, LayersIcon } from 'lucide-mui'

import type { CategoryDialogRequest } from '../types/dialogs'
import { useDebounce } from '@/hooks/useDebounce'
import { useCategoryDialogs } from '../hooks/useCategoryDialogs'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { useGetAllCategoriesQuery, useGetAllCategoryModifiersQuery } from '../categoriesApiSlice'
import { useGetAllCategoryGroupsQuery } from '../categoryGroupsApiSlice'
import { CategoryFilters, type CategoryFilterState } from '../components/CategoryFilters'
import { CategoryTable } from '../components/CategoryTable/index'
import { CategoryDialogs } from '../components/Dialogs/CategoryDialogs'

export const CategoriesView: FC = () => {
	const [filters, setFilters] = useState<CategoryFilterState>({
		group: 'all',
		status: 'all',
		priority: 'all',
		search: '',
	})
	const debouncedSearch = useDebounce(filters.search, 300) as string

	const [createAnchor, setCreateAnchor] = useState<HTMLElement | null>(null)

	const { data: categories } = useGetAllCategoriesQuery()
	const { data: categoryGroups } = useGetAllCategoryGroupsQuery()
	const { data: groups } = useGetAllGroupsQuery()
	const { data: modifiers } = useGetAllCategoryModifiersQuery()

	const resetFilters = () => {
		setFilters({ group: 'all', status: 'all', priority: 'all', search: '' })
	}

	const { dialog, openDialog, closeDialog } = useCategoryDialogs({
		groups: groups?.data || [],
		categories: categories?.data || [],
	})

	const create = (request: CategoryDialogRequest) => {
		setCreateAnchor(null)
		openDialog(request)
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
					<Typography variant='body2' sx={{ color: '#6b7280', mt: 0.25 }}>
						Управление разделами и категориями, их привязкой к группам исполнителей
					</Typography>
				</Box>

				<Button
					variant='outlined'
					sx={{
						borderRadius: '8px',
						textTransform: 'none',
						background: '#fff',
						width: { xs: '100%', sm: 'auto' },
					}}
					onClick={e => setCreateAnchor(e.currentTarget)}
				>
					<PlusIcon color='primary' sx={{ fontSize: 16, mr: 1 }} />
					Создать
				</Button>
				<Popover
					open={Boolean(createAnchor)}
					anchorEl={createAnchor}
					onClose={() => setCreateAnchor(null)}
					anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
					transformOrigin={{ vertical: 'top', horizontal: 'left' }}
					slotProps={{ paper: { sx: { borderRadius: 2.5, minWidth: 210, p: 0.5 } } }}
				>
					<MenuList>
						<MenuItem onClick={() => create({ kind: 'create-category', sectionId: null })}>
							<FolderIcon sx={{ fontSize: 16, mr: 1 }} /> Категорию
						</MenuItem>
						<MenuItem onClick={() => create({ kind: 'create-section' })}>
							<LayersIcon sx={{ fontSize: 16, mr: 1 }} /> Раздел
						</MenuItem>
						<MenuItem onClick={() => create({ kind: 'create-modifier' })}>
							<CornerDownRightIcon sx={{ fontSize: 16, mr: 1 }} /> Подкатегорию
						</MenuItem>
					</MenuList>
				</Popover>
			</Box>

			<CategoryFilters
				groups={groups?.data || []}
				filters={filters}
				onChange={patch => setFilters(prev => ({ ...prev, ...patch }))}
				onReset={resetFilters}
			/>

			<CategoryTable
				categories={categories?.data || []}
				categoryGroups={categoryGroups?.data || []}
				groups={groups?.data || []}
				modifiers={modifiers?.data}
				filters={filters}
				search={debouncedSearch}
				onDialog={openDialog}
			/>
			<CategoryDialogs
				state={dialog}
				groups={groups?.data || []}
				categories={categories?.data || []}
				categoryGroups={categoryGroups?.data || []}
				onClose={closeDialog}
			/>
		</Box>
	)
}
