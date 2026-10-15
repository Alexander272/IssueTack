import { type FC, useCallback, useMemo, useState } from 'react'
import { Box } from '@mui/material'

import { Pagination } from '@/components/Pagination/Pagination'
import { useDebounce } from '@/hooks/useDebounce'
import type { EquipmentStatus } from '../../types/equipment'
import { useGetAllEquipmentQuery, useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { EquipmentTable } from '../Table'
import { EquipmentToolbar } from '../Toolbar'
import { isTreeMode, type EquipmentFilters } from '../Toolbar/types'
import { EquipmentTreeContext } from '../treeState'
import { EquipmentFlatRows } from './EquipmentFlatRows'
import { EquipmentRootRows } from './EquipmentTreeRows'
import { bundleRoots, sortNodes, toggleInSet } from './rows'

const PAGE_SIZE = 50

const DEFAULT_FILTERS: EquipmentFilters = { search: '', typeFilter: 'all', status: 'all' }

type Props = {
	onOpen: (id: string) => void
}

// Контейнер списка оборудования. Дерево грузится лениво (корни — страницей, дети —
// при раскрытии), тип сужает набор корней (единицы/комплекты), а поиск отдаёт плоский
// серверный список. Так браузер не тянет весь справочник: на страницу приходит ровно
// то, что показано.
export const EquipmentList: FC<Props> = ({ onOpen }) => {
	const [filters, setFilters] = useState<EquipmentFilters>(DEFAULT_FILTERS)
	const [page, setPage] = useState(0)
	const [expanded, setExpanded] = useState<Set<string>>(new Set())

	const { search, typeFilter, status } = filters
	const debouncedSearch = useDebounce(search, 300) as string
	const query = debouncedSearch.trim()
	const treeMode = isTreeMode(query)
	const statusFilter = status === 'all' ? undefined : (status as EquipmentStatus)
	// Тип сужает корни дерева; в плоском списке (поиск) — обычный фильтр по is_kit.
	const isKit = typeFilter === 'kit' ? true : typeFilter === 'unit' ? false : undefined

	const rootsQuery = useGetEquipmentTreeQuery(
		{ status: statusFilter, isKit, limit: PAGE_SIZE, offset: page * PAGE_SIZE },
		{ skip: !treeMode },
	)
	const flatQuery = useGetAllEquipmentQuery(
		{
			search: query || undefined,
			status: statusFilter,
			isKit,
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
	const toggleAll = useCallback(
		(expand: boolean) => {
			if (!expand) {
				setExpanded(new Set())
				return
			}
			const ids = roots.filter(n => n.hasChildren).map(n => n.id)
			const keys = bundles.filter(b => b.type === 'group').map(b => b.key)
			setExpanded(prev => new Set([...prev, ...ids, ...keys]))
		},
		[roots, bundles],
	)

	// Строки дерева читают раскрытие/открытие из контекста: протаскивать их через
	// рекурсию TreeNode/NodeChildren не нужно.
	const treeState = useMemo(
		() => ({ tree: treeMode, expanded, toggle, open: onOpen }),
		[treeMode, expanded, toggle, onOpen],
	)

	return (
		<Box>
			<EquipmentToolbar filters={filters} onFiltersChange={handleFiltersChange} onToggleAll={toggleAll} />

			<EquipmentTreeContext.Provider value={treeState}>
				<EquipmentTable isLoading={isLoading} total={total} empty={empty}>
					{treeMode ? <EquipmentRootRows bundles={bundles} /> : <EquipmentFlatRows items={flatItems} />}
				</EquipmentTable>
			</EquipmentTreeContext.Provider>

			{totalPages > 1 && (
				<Box sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}>
					<Pagination page={page + 1} totalPages={totalPages} onClick={p => setPage(p - 1)} />
				</Box>
			)}
		</Box>
	)
}
