import { useCallback, useMemo, useState, type FC } from 'react'
import dayjs from 'dayjs'
import { Box, Grid, Typography } from '@mui/material'
import { useSearchParams } from 'react-router'

import { useAppSelector } from '@/hooks/redux'
import { getIsManager } from '@/features/user/userSlice'
import { PeriodPicker } from '@/components/Period/Period'
import { getPresetDates } from '@/components/Period/utils'
import type { DateRange } from '@/components/Period/types'
import { useGetTicketStatisticsQuery } from '../statisticsApiSlice'
import { KpiCards } from '../components/KpiCards'
import { TrendChart } from '../components/TrendChart'
import { StatusChart } from '../components/StatusChart'
import { BreakdownChart } from '../components/BreakdownChart'
import { NestedDonut, type IDonutBucket } from '../components/NestedDonut'
import { DEFAULT_TOP_N } from '../components/TopNSelect'
import { StatisticsFilters } from '../components/StatisticsFilters'
import type { IStatisticsFilter, IStatisticsRefinements } from '../types/statistics'

// refinementKeys — имена query-параметров URL, в которых хранится состояние панели
// фильтров. Повторяющиеся параметры (?assigneeId=a&assigneeId=b) сохраняют
// мультивыбор и делают ссылку на отфильтрованный вид шарящейся.
const refinementKeys: (keyof IStatisticsRefinements)[] = ['assigneeId', 'categoryId', 'groupId', 'siteId']

export const StatisticsView: FC = () => {
	const isManager = useAppSelector(getIsManager)
	const [range, setRange] = useState<DateRange>(() => getPresetDates('month', 'month', 1))
	// Сколько исполнителей/заказчиков показывать в донат-диаграммах; хвост
	// сворачивается в «Прочие» (см. NestedDonut).
	const [topNAssignee, setTopNAssignee] = useState(DEFAULT_TOP_N)
	const [topNOwner, setTopNOwner] = useState(DEFAULT_TOP_N)
	const [searchParams, setSearchParams] = useSearchParams()

	const refinements = useMemo<IStatisticsRefinements>(
		() => ({
			assigneeId: searchParams.getAll('assigneeId'),
			categoryId: searchParams.getAll('categoryId'),
			groupId: searchParams.getAll('groupId'),
			siteId: searchParams.getAll('siteId'),
		}),
		[searchParams],
	)

	const updateRefinements = useCallback(
		(patch: Partial<IStatisticsRefinements>) => {
			const next = new URLSearchParams(searchParams)
			for (const [key, ids] of Object.entries(patch) as [keyof IStatisticsRefinements, string[]][]) {
				next.delete(key)
				;(ids ?? []).forEach(id => next.append(key, id))
			}
			setSearchParams(next, { replace: true })
		},
		[searchParams, setSearchParams],
	)

	const resetRefinements = useCallback(() => {
		const next = new URLSearchParams(searchParams)
		refinementKeys.forEach(key => next.delete(key))
		setSearchParams(next, { replace: true })
	}, [searchParams, setSearchParams])

	const query = useMemo<IStatisticsFilter>(() => {
		const params: IStatisticsFilter = {
			from: dayjs(range.startDate).format('YYYY-MM-DD'),
			to: dayjs(range.endDate).format('YYYY-MM-DD'),
		}
		refinementKeys.forEach(key => {
			if (refinements[key].length) params[key] = refinements[key]
		})
		return params
	}, [range, refinements])

	const { data, isFetching } = useGetTicketStatisticsQuery(query)
	const stats = data?.data

	const assigneeBuckets = useMemo<IDonutBucket[]>(
		() =>
			(stats?.workload ?? []).map(item => ({
				id: item.userId,
				name: item.name,
				total: item.total,
				active: item.active,
			})),
		[stats?.workload],
	)
	const ownerBuckets = useMemo<IDonutBucket[]>(() => stats?.byOwner ?? [], [stats?.byOwner])

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto', p: 3 }}>
			<Box
				sx={{
					display: 'flex',
					flexWrap: 'wrap',
					flexDirection: { xs: 'column', sm: 'row' },
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: { xs: 2, sm: 1 },
					mb: 4,
				}}
			>
				<Box sx={{ flexGrow: { xs: 0, sm: 1 } }}>
					<Typography variant='h5' sx={{ fontWeight: 'bold', color: '#1f2937' }}>
						Статистика
					</Typography>
					<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
						Показатели заявок за выбранный период
					</Typography>
				</Box>
				<PeriodPicker value={range} onChange={setRange} />
				<StatisticsFilters
					value={refinements}
					onChange={updateRefinements}
					onReset={resetRefinements}
					isManager={isManager}
				/>
			</Box>

			<KpiCards summary={stats?.summary} isLoading={isFetching} />

			<Grid container spacing={3} sx={{ mb: 3 }}>
				<Grid size={{ xs: 12, lg: 8 }}>
					<TrendChart points={stats?.trend ?? []} isLoading={isFetching} />
				</Grid>
				<Grid size={{ xs: 12, lg: 4 }}>
					<StatusChart buckets={stats?.byStatus ?? []} isLoading={isFetching} />
				</Grid>
			</Grid>

			<Grid container spacing={3} sx={{ mb: isManager ? 3 : 0 }}>
				{/* Менеджеру три в ряд с 900px (md), не-менеджеру две по 6; lg всплывает из md. */}
				<Grid size={{ xs: 12, md: isManager ? 4 : 6 }}>
					<BreakdownChart title='По категориям' buckets={stats?.byCategory ?? []} isLoading={isFetching} />
				</Grid>
				<Grid size={{ xs: 12, md: isManager ? 4 : 6 }}>
					<BreakdownChart
						title='По площадкам'
						buckets={stats?.bySite ?? []}
						color='#27b05c'
						isLoading={isFetching}
					/>
				</Grid>
				{isManager && (
					<Grid size={{ xs: 12, md: 4 }}>
						<BreakdownChart
							title='По группам'
							buckets={stats?.byGroup ?? []}
							color='#ff9800'
							isLoading={isFetching}
						/>
					</Grid>
				)}
			</Grid>

			{isManager && (
				<Grid container spacing={3}>
					<Grid size={{ xs: 12, lg: 6 }}>
						<NestedDonut
							title='Нагрузка исполнителей'
							buckets={assigneeBuckets}
							topN={topNAssignee}
							onTopNChange={setTopNAssignee}
							isLoading={isFetching}
						/>
					</Grid>
					<Grid size={{ xs: 12, lg: 6 }}>
						<NestedDonut
							title='Задачи от заказчиков'
							buckets={ownerBuckets}
							topN={topNOwner}
							onTopNChange={setTopNOwner}
							isLoading={isFetching}
						/>
					</Grid>
				</Grid>
			)}
		</Box>
	)
}
