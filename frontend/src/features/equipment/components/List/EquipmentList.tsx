import { type FC, useCallback, useMemo, useState } from 'react'
import { Box } from '@mui/material'

import { Pagination } from '@/components/Pagination/Pagination'
import { useDebounce } from '@/hooks/useDebounce'
import type { EquipmentStatus } from '../../types/equipment'
import { useGetAllEquipmentQuery, useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { EquipmentTable } from '../Table'
import { EquipmentToolbar } from '../Toolbar'
import type { EquipmentFilters } from '../Toolbar/types'
import { EquipmentFlatRows } from './EquipmentFlatRows'
import { EquipmentRootRows } from './EquipmentTreeRows'
import { bundleRoots, isTreeMode, sortNodes, toggleInSet } from './rows'

const PAGE_SIZE = 50

const DEFAULT_FILTERS: EquipmentFilters = { search: '', typeFilter: 'all', status: 'all' }

type Props = {
	onOpen: (id: string) => void
}

// Контейнер списка оборудования. Дерево грузится лениво (корни — страницей, дети —
// при раскрытии), поиск и фильтр по типу — плоский серверный список. Так браузер не
// тянет весь справочник: на страницу приходит ровно то, что показано.
export const EquipmentList: FC<Props> = ({ onOpen }) => {
	const [filters, setFilters] = useState<EquipmentFilters>(DEFAULT_FILTERS)
	const [page, setPage] = useState(0)
	const [expanded, setExpanded] = useState<Set<string>>(new Set())

	const { search, typeFilter, status } = filters
	const debouncedSearch = useDebounce(search, 300) as string
	const query = debouncedSearch.trim()
	const treeMode = isTreeMode(query, typeFilter)
	const statusFilter = status === 'all' ? undefined : (status as EquipmentStatus)

	const rootsQuery = useGetEquipmentTreeQuery(
		{ status: statusFilter, limit: PAGE_SIZE, offset: page * PAGE_SIZE },
		{ skip: !treeMode },
	)
	const flatQuery = useGetAllEquipmentQuery(
		{
			search: query || undefined,
			status: statusFilter,
			isKit: typeFilter === 'kit' ? true : typeFilter === 'unit' ? false : undefined,
			limit: PAGE_SIZE,
			offset: page * PAGE_SIZE,
		},
		{ skip: treeMode },
	)

	const active = treeMode ? rootsQuery : flatQuery
	const total = active.data?.total ?? 0
	const isLoading = active.isFetching
	const totalPages = Math.ceil(total / PAGE_SIZE) || 1

	const roots = useMemo(() => rootsQuery.data?.data ?? [], [rootsQuery.data])
	const items = useMemo(() => flatQuery.data?.data ?? [], [flatQuery.data])

	const bundles = useMemo(() => bundleRoots(sortNodes(roots)), [roots])
	const flatItems = useMemo(() => sortNodes(items), [items])
	const empty = treeMode ? bundles.length === 0 : items.length === 0

	// Обёрнуто в useCallback, чтобы ссылка на функцию не менялась и тулбар не
	// перерисовывался лишний раз.
	const handleFiltersChange = useCallback((updates: Partial<EquipmentFilters>) => {
		setFilters(prev => ({ ...prev, ...updates }))
		setPage(0)
		setExpanded(new Set())
	}, [])

	const toggle = useCallback((id: string) => setExpanded(prev => toggleInSet(prev, id)), [])

	// «Развернуть всё» раскрывает корни текущей страницы (глубже — по клику):
	// развернуть ленивое дерево целиком нельзя, не загрузив весь справочник.
	const expandAll = useCallback(() => {
		const ids = roots.filter(n => n.hasChildren).map(n => n.id)
		const keys = bundles.filter(b => b.type === 'group').map(b => b.key)
		setExpanded(prev => new Set([...prev, ...ids, ...keys]))
	}, [roots, bundles])

	const collapseAll = useCallback(() => setExpanded(new Set()), [])

	return (
		<Box>
			<EquipmentToolbar
				filters={filters}
				onFiltersChange={handleFiltersChange}
				treeMode={treeMode}
				onExpandAll={expandAll}
				onCollapseAll={collapseAll}
			/>

			<EquipmentTable isLoading={isLoading} total={total} empty={empty}>
				{treeMode ? (
					<EquipmentRootRows bundles={bundles} expanded={expanded} onToggle={toggle} onOpen={onOpen} />
				) : (
					<EquipmentFlatRows items={flatItems} onOpen={onOpen} />
				)}
			</EquipmentTable>

			{totalPages > 1 && (
				<Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
					<Pagination page={page + 1} totalPages={totalPages} onClick={p => setPage(p - 1)} />
				</Box>
			)}
		</Box>
	)
}
