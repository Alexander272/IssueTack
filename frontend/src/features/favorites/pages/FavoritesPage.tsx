import { useState, useCallback, useMemo, useEffect } from 'react'
import { Box, Typography, Tab, Tabs } from '@mui/material'
import { useNavigate } from 'react-router'
import { Pin, Star } from 'lucide-mui'

import type { GroupByField } from '@/features/tasks/constants/taskMaps'
import type { FilterValues } from '@/features/tasks/components/filters'
import type { ITask, ITaskFilter } from '@/features/tasks/types/task'
import { AppRoutes } from '@/pages/router/routes'
import { useGetTasksQuery } from '@/features/tasks/tasksApiSlice'
import { TaskFilters } from '@/features/tasks/components/filters'
import { TaskListView } from '@/features/tasks/components/Table'

type FavoriteTab = 'temporary' | 'permanent'

const DEFAULT_FILTERS: FilterValues = {
	sort: 'dueDate_asc',
	search: '',
	groupBy: 'site',
	groupEnabled: true,

	ticketNumber: undefined,
	ownerId: undefined,
	siteIds: undefined,
	dueDateFrom: undefined,
	dueDateTo: undefined,
	priorities: undefined,
	assigneeId: undefined,
	statuses: undefined,
}

export const FavoritesPage = () => {
	const navigate = useNavigate()
	const rowsPerPage = 20

	const STORAGE_KEY = '@issueTrack/favoritesFilters'

	const loadFilters = (): FilterValues => {
		try {
			const saved = localStorage.getItem(STORAGE_KEY)
			if (saved) return { ...DEFAULT_FILTERS, ...JSON.parse(saved) }
		} catch {
			/* ignore */
		}
		return DEFAULT_FILTERS
	}

	const [filters, setFilters] = useState<FilterValues>(loadFilters)
	const [page, setPage] = useState(0)
	const [tab, setTab] = useState<FavoriteTab>('temporary')

	useEffect(() => {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(filters))
	}, [filters, STORAGE_KEY])

	const isArchive = tab === 'permanent'

	const queryFilter: ITaskFilter = useMemo(
		() => ({
			number: filters.ticketNumber ? Number(filters.ticketNumber) : undefined,
			ownerId: filters.ownerId ?? undefined,
			assigneeId: filters.assigneeId ?? undefined,
			siteIds: filters.siteIds?.length ? filters.siteIds : undefined,
			priorities: filters.priorities?.length ? filters.priorities : undefined,
			statuses: filters.statuses?.length ? filters.statuses : undefined,
			dueDateFrom: filters.dueDateFrom || undefined,
			dueDateTo: filters.dueDateTo || undefined,
			search: filters.search || undefined,
			sort: filters.sort,
			favoritesType: tab,
			archived: isArchive || undefined,
			limit: isArchive ? rowsPerPage : undefined,
			offset: isArchive ? page * rowsPerPage : undefined,
		}),
		[filters, tab, page, isArchive],
	)

	const { data, isFetching } = useGetTasksQuery(queryFilter)

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const totalPages = isArchive ? Math.ceil(total / rowsPerPage) || 1 : 1

	const handleFilterChange = useCallback((patch: Partial<FilterValues>) => {
		setFilters(prev => ({ ...prev, ...patch }))
		setPage(0)
	}, [])

	const handleReset = useCallback(() => {
		setFilters(DEFAULT_FILTERS)
		setPage(0)
		localStorage.removeItem(STORAGE_KEY)
	}, [STORAGE_KEY])

	const handleTaskClick = useCallback(
		(task: ITask) => {
			navigate(`${AppRoutes.Tasks}/${task.id}`)
		},
		[navigate],
	)

	const handlePageChange = useCallback((p: number) => setPage(p - 1), [])

	return (
		<Box sx={{ p: 3 }}>
			<Box sx={{ mb: 1 }}>
				<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
					Избранное
				</Typography>
				<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
					Закреплённые и отмеченные звёздочкой заявки
				</Typography>
			</Box>

			<Tabs
				value={tab}
				onChange={(_, v: FavoriteTab) => {
					setTab(v)
					setPage(0)
				}}
				sx={{
					mb: 2,
					borderBottom: '1px solid #e5e7eb',
					minHeight: 36,
					'& .MuiTab-root': { minHeight: 36, py: 0, textTransform: 'none' },
				}}
			>
				<Tab value='temporary' label='Закреплённые' iconPosition='start' icon={<Pin sx={{ fontSize: 14 }} />} />
				<Tab value='permanent' label='Избранные' iconPosition='start' icon={<Star sx={{ fontSize: 14 }} />} />
			</Tabs>

			<TaskFilters filters={filters} onChange={handleFilterChange} onReset={handleReset} hideGrouping={isArchive} />

			<TaskListView
				tasks={tasks}
				isFetching={isFetching}
				total={total}
				isArchive={isArchive}
				groupBy={filters.groupBy as GroupByField}
				groupEnabled={!isArchive && filters.groupEnabled}
				onTaskClick={handleTaskClick}
				sort={filters.sort}
				onSortChange={sort => handleFilterChange({ sort })}
				page={page}
				totalPages={totalPages}
				onPageChange={handlePageChange}
			/>
		</Box>
	)
}
