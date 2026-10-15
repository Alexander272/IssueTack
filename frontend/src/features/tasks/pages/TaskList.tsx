import { useState, useCallback, useMemo, useEffect } from 'react'
import { Box, Button, Typography, Tab, Tabs, useTheme } from '@mui/material'
import { useNavigate, useSearchParams } from 'react-router'
import { ArchiveIcon, InboxIcon, PlusIcon } from 'lucide-mui'

import type { GroupByField } from '../constants/taskMaps'
import { statusOptionsFor } from '../constants/taskMaps'
import type { FilterValues } from '../components/filters'
import type { ITask, ITaskFilter } from '../types/task'
import { AppRoutes } from '@/pages/router/routes'
import { PermRules } from '@/features/access/constants/permissions'
import { useAppSelector } from '@/hooks/redux'
import { useCan } from '@/features/access/utils/can'
import { useGetTasksQuery } from '../tasksApiSlice'
import { getIsManager } from '@/features/user/userSlice'
import { useDebounce } from '@/hooks/useDebounce'
import { TaskFilters } from '../components/filters'
import { TaskListView } from '../components/Table'
import { TaskCreateModal } from '../components/TaskCreateModal'

type Mode = 'created' | 'assigned'
type Tab = 'active' | 'archive'

type Props = {
	mode?: Mode
}

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
	equipmentId: undefined,
}

// UUID-параметр приходит только из ссылки (из UI такой фильтр не выбрать), то есть
// может быть подправлен руками: битое значение иначе уходит на бэкенд и падает
// разбором uuid в 400 с несбрасываемым тостом. Такое значение молча игнорируем.
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

const PAGE_TITLE: Record<Mode, string> = {
	created: 'Заявки',
	assigned: 'Мои задачи',
}

const PAGE_DESC: Record<Mode, string> = {
	created: 'Созданные вами заявки',
	assigned: 'Задачи, назначенные лично или группам',
}

export const TaskList = ({ mode = 'created' }: Props) => {
	const { palette } = useTheme()
	const navigate = useNavigate()
	const [searchParams] = useSearchParams()
	const canCreate = useCan(PermRules.Tasks.Write)
	const isManager = useAppSelector(getIsManager)
	const rowsPerPage = 20

	// Фильтр по оборудованию приходит ссылкой (?equipmentId=…, кнопка «Показать все»
	// в карточке единицы) и живёт в адресе, а не в localStorage: сохранённый фильтр
	// с прошлого визита к другому оборудованию пользователю не нужен. Название для
	// подписи чипа тоже едет ссылкой (equipmentName) — отдельный запрос за единицей
	// ради подписи тянул бы за собой право equipment:read, которого у читателя
	// списка может не быть.
	const rawEquipmentId = searchParams.get('equipmentId')
	const urlEquipmentId = rawEquipmentId && UUID_RE.test(rawEquipmentId) ? rawEquipmentId : undefined
	const urlEquipmentName = searchParams.get('equipmentName') ?? undefined

	// Режим списка: «Показать все» из карточки оборудования ведёт сюда за всеми
	// заявками единицы, а страница по умолчанию ограничивает выборку своим режимом
	// (mode=created — «созданные вами»). При активном фильтре по оборудованию режим
	// не отправляем, чтобы список совпадал с блоком в карточке. Обход ничего не
	// открывает сверх доступа: без режима бэкенд по-прежнему отдаёт только заявки
	// своих групп и внегрупповые на себя, а начальнику области — его реалм.
	const listMode: Mode | undefined = urlEquipmentId ? undefined : mode
	const statusMode = listMode === 'assigned' ? 'assigned' : 'default'

	// Заголовок и подпись при активном фильтре по оборудованию тоже меняются:
	// «Заявки — Созданные вами заявки» над чужими заявками вводили бы в заблуждение.
	const title = urlEquipmentId ? 'Заявки по оборудованию' : PAGE_TITLE[mode]
	const desc = urlEquipmentId
		? (urlEquipmentName ?? 'Все заявки по этой единице, доступные вам')
		: isManager && mode === 'created'
			? 'Все доступные заявки'
			: PAGE_DESC[mode]

	const STORAGE_KEY = `@issueTrack/taskFilters_${mode}`

	// clampStatuses — фильтр по статусам должен уметься во вкладку: сохранённые
	// в localStorage статусы, которых на текущей вкладке уже нет (например «Решены»
	// в «Мои задачи» после переноса в архив), выкидываем — иначе висящий чип
	// в фильтре даёт всегда пустой список.
	const clampStatuses = (f: FilterValues, isArchiveTab: boolean): FilterValues => {
		const allowed = new Set(statusOptionsFor(isArchiveTab, statusMode))
		const statuses = f.statuses?.filter(s => allowed.has(s))
		if (statuses === undefined) return f
		if (statuses.length === (f.statuses?.length ?? -1)) return f
		return { ...f, statuses }
	}

	const loadFilters = (): FilterValues => {
		try {
			const saved = localStorage.getItem(STORAGE_KEY)
			if (saved) return clampStatuses({ ...DEFAULT_FILTERS, ...JSON.parse(saved) }, false)
		} catch {
			/* ignore */
		}
		return DEFAULT_FILTERS
	}

	const [filters, setFilters] = useState<FilterValues>(() => {
		const base = loadFilters()
		return urlEquipmentId ? { ...base, equipmentId: urlEquipmentId } : base
	})
	const [createOpen, setCreateOpen] = useState(false)
	const [page, setPage] = useState(0)
	const [tab, setTab] = useState<Tab>('active')

	// Ссылку на другое оборудование можно открыть поверх уже смонтированного списка
	// (возврат по истории к ссылке с оборудованием) — адрес остаётся источником правды.
	// Синхронизируем состояние прямо в рендере, а не эффектом: это штатный способ
	// подстроить state под изменившийся вход, без лишней отрисовки.
	const [syncedEquipmentId, setSyncedEquipmentId] = useState(urlEquipmentId)
	if (syncedEquipmentId !== urlEquipmentId) {
		setSyncedEquipmentId(urlEquipmentId)
		setFilters(prev => (prev.equipmentId === urlEquipmentId ? prev : { ...prev, equipmentId: urlEquipmentId }))
		setPage(0)
	}

	// В localStorage едет всё, кроме фильтра по оборудованию: он приходит ссылкой
	// и не должен «прилипать» к списку после ухода со страницы.
	useEffect(() => {
		const saved: Partial<FilterValues> = { ...filters }
		delete saved.equipmentId
		localStorage.setItem(STORAGE_KEY, JSON.stringify(saved))
	}, [filters, STORAGE_KEY])

	const isArchive = tab === 'archive'

	// Статусы, доступные на текущей вкладке: на «Активных» нет закрытых/отменённых,
	// на «Мои задачи» (assigned) — и решённых (они ушли в архив).
	const statusOptions = useMemo(
		() => statusOptionsFor(isArchive, statusMode),
		[isArchive, statusMode],
	)

	// Сетевой запрос дебаунсится, чтобы не долбить бэкенд на каждое нажатие
	// клавиши в поиске; сам инпут фильтров остаётся мгновенным.
	const debouncedSearch = useDebounce(filters.search, 300) as string

	const numberFilter = filters.ticketNumber ? Number(filters.ticketNumber) : NaN

	const queryFilter: ITaskFilter = useMemo(
		() => ({
			number: Number.isNaN(numberFilter) ? undefined : numberFilter,
			ownerId: filters.ownerId ?? undefined,
			assigneeId: filters.assigneeId ?? undefined,
			siteIds: filters.siteIds?.length ? filters.siteIds : undefined,
			priorities: filters.priorities?.length ? filters.priorities : undefined,
			statuses: filters.statuses?.length ? filters.statuses : undefined,
			dueDateFrom: filters.dueDateFrom || undefined,
			dueDateTo: filters.dueDateTo || undefined,
			equipmentId: filters.equipmentId || undefined,
			search: debouncedSearch || undefined,
			sort: filters.sort,
			mode: listMode,
			archived: isArchive || undefined,
			limit: isArchive ? rowsPerPage : undefined,
			offset: isArchive ? page * rowsPerPage : undefined,
		}),
		[filters, listMode, page, isArchive, numberFilter, debouncedSearch],
	)

	const { data, isFetching } = useGetTasksQuery(queryFilter)

	// Название единицы для чипа фильтра: едет в ссылке, чтобы не тянуть запрос.
	const equipmentLabel = urlEquipmentId ? urlEquipmentName : undefined

	const tasks = data?.data ?? []
	const total = data?.total ?? 0

	const totalPages = isArchive ? Math.ceil(total / rowsPerPage) || 1 : 1

	// Фильтр по оборудованию — единственный, что пришёл ссылкой: снятие чипа должно
	// убрать и параметр адреса, иначе возврат к ссылке/перезагрузка снова его включит.
	// Проверяем наличие в адресе (а не принятый фильтр): так же чистится и битый параметр.
	const dropEquipmentParam = useCallback(() => {
		if (!searchParams.has('equipmentId') && !searchParams.has('equipmentName')) return
		const next = new URLSearchParams(searchParams)
		next.delete('equipmentId')
		next.delete('equipmentName')
		navigate({ search: next.toString() }, { replace: true })
	}, [navigate, searchParams])

	const handleFilterChange = useCallback(
		(patch: Partial<FilterValues>) => {
			setFilters(prev => ({ ...prev, ...patch }))
			setPage(0)
			if ('equipmentId' in patch && !patch.equipmentId) dropEquipmentParam()
		},
		[dropEquipmentParam],
	)

	const handleReset = useCallback(() => {
		setFilters(DEFAULT_FILTERS)
		setPage(0)
		localStorage.removeItem(STORAGE_KEY)
		dropEquipmentParam()
	}, [STORAGE_KEY, dropEquipmentParam])

	const handleTaskClick = useCallback(
		(task: ITask) => {
			navigate(`${AppRoutes.Tasks}/${task.id}`)
		},
		[navigate],
	)

	const handlePageChange = useCallback((p: number) => setPage(p - 1), [])

	return (
		<Box sx={{ p: 3 }}>
			<Box
				sx={{
					display: 'flex',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 0 },
					mb: 1,
				}}
			>
				<Box>
					<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
						{title}
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						{desc}
					</Typography>
				</Box>
				{mode === 'created' && canCreate && (
					<Button
						variant='outlined'
						sx={{
							borderRadius: '8px',
							textTransform: 'none',
							background: '#fff',
							width: { xs: '100%', sm: 'auto' },
						}}
						onClick={() => setCreateOpen(true)}
					>
						<PlusIcon sx={{ color: palette.primary.main, fontSize: 16, mr: 1.5 }} />
						Создать заявку
					</Button>
				)}
			</Box>

			<Tabs
				value={tab}
				onChange={(_, v: Tab) => {
					setTab(v)
					setPage(0)
					setFilters(prev => clampStatuses(prev, v === 'archive'))
				}}
				sx={{
					mb: 2,
					borderBottom: '1px solid #e5e7eb',
					minHeight: 36,
					'& .MuiTab-root': { minHeight: 36, py: 0, textTransform: 'none' },
				}}
			>
				<Tab
					value='active'
					label='Активные задачи'
					iconPosition='start'
					icon={<InboxIcon sx={{ fontSize: 14 }} />}
				/>
				<Tab value='archive' label='Архив' iconPosition='start' icon={<ArchiveIcon sx={{ fontSize: 14 }} />} />
			</Tabs>

			<TaskFilters
				filters={filters}
				onChange={handleFilterChange}
				onReset={handleReset}
				hideGrouping={isArchive}
				statusOptions={statusOptions}
				equipmentLabel={equipmentLabel}
			/>

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

			<TaskCreateModal open={createOpen} onClose={() => setCreateOpen(false)} />
		</Box>
	)
}
