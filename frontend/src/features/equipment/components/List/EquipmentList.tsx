import { type FC, useCallback, useMemo, useState } from 'react'
import { Box, useMediaQuery, useTheme } from '@mui/material'

import type { EquipmentStatus } from '../../types/equipment'
import { isTreeMode, type EquipmentFilters, type TypeFilter } from '../Toolbar/types'
import { useDebounce } from '@/hooks/useDebounce'
import { useGetAllEquipmentQuery, useGetEquipmentTreeQuery } from '../../equipmentApiSlice'
import { Pagination } from '@/components/Pagination/Pagination'
import { EquipmentTable } from '../Table'
import { EquipmentToolbar } from '../Toolbar'
import { EquipmentTreeContext } from '../treeState'
import { EquipmentFlatRows } from './EquipmentFlatRows'
import { EquipmentRootRows } from './EquipmentTreeRows'
import { bundleRoots, sortNodes, toggleInSet } from './rows'

const PAGE_SIZE = 50

const DEFAULT_FILTERS: EquipmentFilters = {
	search: '',
	typeFilter: 'all',
	status: 'all',
	siteId: null,
	locationId: null,
}

// Тип фильтра → is_kit для запроса: не задан («all») — не фильтровать, единицы / комплекты — только свои.
const IS_KIT_BY_TYPE: Record<TypeFilter, boolean | undefined> = { all: undefined, unit: false, kit: true }

type Props = {
	onOpen: (id: string) => void
}

// Контейнер списка оборудования. Дерево грузится лениво (корни — страницей, дети —
// при раскрытии), тип сужает набор корней (единицы/комплекты), а поиск отдаёт плоский
// серверный список. Так браузер не тянет весь справочник: на страницу приходит ровно
// то, что показано.
export const EquipmentList: FC<Props> = ({ onOpen }) => {
	const theme = useTheme()
	// Узкий экран — строки-таблицы заменяются карточками. Считаем один раз здесь и
	// кладём в контекст: строки дерева получают раскладку без своих media query.
	const mobile = useMediaQuery(theme.breakpoints.down('md'))
	const [filters, setFilters] = useState<EquipmentFilters>(DEFAULT_FILTERS)
	const [page, setPage] = useState(0)
	const [expanded, setExpanded] = useState<Set<string>>(new Set())

	const { search, typeFilter, status, siteId, locationId } = filters
	const debouncedSearch = useDebounce(search, 300)
	const query = debouncedSearch.trim()
	const treeMode = isTreeMode(query, Boolean(siteId || locationId))
	const statusFilter = status === 'all' ? undefined : (status as EquipmentStatus)
	// Тип сужает корни дерева; в плоском списке (поиск) — обычный фильтр по is_kit.
	const isKit = IS_KIT_BY_TYPE[typeFilter]

	const rootsQuery = useGetEquipmentTreeQuery(
		{ status: statusFilter, isKit, search: query || undefined, limit: PAGE_SIZE, offset: page * PAGE_SIZE },
		{ skip: !treeMode },
	)
	const flatQuery = useGetAllEquipmentQuery(
		{
			search: query || undefined,
			status: statusFilter,
			isKit,
			siteId: siteId ?? undefined,
			locationId: locationId ?? undefined,
			limit: PAGE_SIZE,
			offset: page * PAGE_SIZE,
		},
		{ skip: treeMode },
	)

	const active = treeMode ? rootsQuery : flatQuery
	const loadedTotal = active.data?.total
	// Пока у нового ключа запроса нет данных (data === undefined), total === 0 — не обнуляем страницу.
	const total = loadedTotal ?? 0
	const isLoading = active.isFetching
	const totalPages = Math.ceil(total / PAGE_SIZE) || 1
	// Derived safePage: если данных ещё нет — оставляем page, иначе ограничиваем последней страницей.
	const safePage = loadedTotal === undefined ? page : Math.min(page, totalPages - 1)

	// Неактивный запрос пропущен (skip), но его данные остаются в кэше RTK Query после
	// переключения режима. Берём данные только активного, чтобы его мемо не пересчитывались
	// от чужих обновлений, и не создаём лишний useMemo ради `data?.data ?? []`.
	const rootsData = treeMode ? rootsQuery.data?.data : undefined
	const itemsData = treeMode ? undefined : flatQuery.data?.data

	const bundles = useMemo(() => bundleRoots(sortNodes(rootsData ?? [])), [rootsData])
	const flatItems = useMemo(() => sortNodes(itemsData ?? []), [itemsData])
	const empty = treeMode ? bundles.length === 0 : flatItems.length === 0

	// Сброс страницы и раскрытых узлов при смене фильтров.
	const handleFiltersChange = useCallback((updates: Partial<EquipmentFilters>) => {
		setFilters(prev => ({ ...prev, ...updates }))
		setPage(0)
		setExpanded(new Set())
	}, [])

	// Сброс страницы и раскрытых узлов при смене страницы пагинации.
	const handlePageChange = useCallback((p: number) => {
		setPage(p - 1)
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
			const ids = (rootsData ?? []).filter(n => n.hasChildren).map(n => n.id)
			const keys = bundles.filter(b => b.type === 'group').map(b => b.key)
			setExpanded(prev => new Set([...prev, ...ids, ...keys]))
		},
		[rootsData, bundles],
	)

	// Строки дерева читают раскрытие/открытие из контекста: протаскивать их через
	// рекурсию TreeNode/NodeChildren не нужно.
	const treeState = useMemo(
		() => ({ tree: treeMode, mobile, expanded, toggle, open: onOpen, search: query }),
		[treeMode, mobile, expanded, toggle, onOpen, query],
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
					<Pagination page={safePage + 1} totalPages={totalPages} onClick={handlePageChange} />
				</Box>
			)}
		</Box>
	)
}
