import { useMemo, type FC } from 'react'
import { Box, Paper, Skeleton, Typography } from '@mui/material'
import { PieChart } from '@mui/x-charts/PieChart'
import type { DefaultizedPieValueType, PieItemIdentifier, PieValueType } from '@mui/x-charts'

import { getAvatarColor } from '@/utils/avatar'
import { getChartColorFromText } from '../utils/color'
import { TopNSelect } from './TopNSelect'

// IDonutBucket — то, что нужно двухкольцевой диаграмме: внутреннее кольцо —
// все заявки сущности (total), внешнее — активные и неактивные (сумма = total).
export interface IDonutBucket {
	id: string
	name: string
	total: number
	active: number
}

// DonutRing — какое кольцо нажали: пусто — внутреннее (все заявки), active и
// closed — половины внешнего кольца. Значения совпадают с параметром `ring` API.
export type DonutRing = '' | 'active' | 'closed'

interface Props {
	title: string
	buckets: IDonutBucket[]
	topN: number
	onTopNChange: (value: number) => void
	isLoading: boolean
	// Клик по сектору: id сущности и кольцо. «Прочие» не кликается.
	onSliceClick?: (bucketId: string, ring: DonutRing) => void
}

const REST_ID = '__rest'
const REST_HEX = '#9aa5b1'
const ZERO_UUID = '00000000-0000-0000-0000-000000000000'

// «Прочие» и заявка без заказчика (zero-uuid) — не человек, поэтому без нормализации
// насыщенности до 75%: иначе серо-голубой #9aa5b1 превращался в яркий #207cdf,
// неотличимый от настоящих сегментов. Берём приглушённое семейство той же базы.
const isNeutral = (id: string) => id === REST_ID || id === ZERO_UUID
const REST_TOTAL = getChartColorFromText(REST_HEX, 14, 72)
const REST_INACTIVE = getChartColorFromText(REST_HEX, 14, 90)

// Три оттенка одного тона: плотность показывает, какая часть кольца активна.
// Активный цвет — тот, что в легенде. Для обычных сущностей база — цвет аватара,
// детерминированный по id: не меняется при смене периода или Top-N и совпадает
// с аватаром пользователя в других местах приложения.
const totalColor = (id: string) => (isNeutral(id) ? REST_TOTAL : getChartColorFromText(getAvatarColor(id), 70, 65))
const activeColor = (id: string) => (isNeutral(id) ? REST_HEX : getChartColorFromText(getAvatarColor(id)))
const inactiveColor = (id: string) =>
	isNeutral(id) ? REST_INACTIVE : getChartColorFromText(getAvatarColor(id), 45, 88)

const sum = (buckets: IDonutBucket[], key: 'total' | 'active') => buckets.reduce((acc, bucket) => acc + bucket[key], 0)

// aggregate сортирует сущности по числу заявок, оставляет topN первых и сворачивает
// хвост в «Прочие». Раньше topN <= 0 означал «Все» — опцию убрали (донат на сотнях
// секторов нечитаем), ветка осталась как защита.
const aggregate = (buckets: IDonutBucket[], topN: number): IDonutBucket[] => {
	const sorted = [...buckets].sort((a, b) => b.total - a.total)
	const shown = topN > 0 ? sorted.slice(0, topN) : sorted
	const rest = sorted.slice(shown.length)
	if (rest.length === 0) return shown
	return [...shown, { id: REST_ID, name: 'Прочие', total: sum(rest, 'total'), active: sum(rest, 'active') }]
}

export const NestedDonut: FC<Props> = ({ title, buckets, topN, onTopNChange, isLoading, onSliceClick }) => {
	const rows = useMemo(() => aggregate(buckets, topN), [buckets, topN])

	// id сектора — `inner-<bucketId>` / `active-<bucketId>` / `inactive-<bucketId>`.
	// Разбираем по первому дефису: сам id — uuid (дефисы внутри) или `__rest`.
	// «Прочие» не кликаются — за ними нет одной сущности.
	const handleItemClick = (_event: unknown, _identifier: PieItemIdentifier, item: DefaultizedPieValueType) => {
		if (!onSliceClick) return
		const raw = `${item.id ?? ''}`
		const sep = raw.indexOf('-')
		if (sep < 0) return
		const bucketId = raw.slice(sep + 1)
		if (bucketId === REST_ID) return
		const prefix = raw.slice(0, sep)
		const ring: DonutRing = prefix === 'active' ? 'active' : prefix === 'inactive' ? 'closed' : ''
		onSliceClick(bucketId, ring)
	}

	// Оба кольца строятся в одном порядке элементов и с paddingAngle: 0 — иначе
	// границы секторов внешнего кольца уехали бы относительно внутреннего.
	const innerSeries = useMemo<PieValueType[]>(
		() =>
			rows.map(row => ({
				id: `inner-${row.id}`,
				value: row.total,
				label: `${row.name}: всего`,
				color: totalColor(row.id),
			})),
		[rows],
	)

	const outerSeries = useMemo<PieValueType[]>(
		() =>
			rows.flatMap(row => {
				const inactive = Math.max(row.total - row.active, 0)
				return [
					{
						id: `active-${row.id}`,
						value: row.active,
						label: `${row.name}: активных`,
						color: activeColor(row.id),
					},
					{
						id: `inactive-${row.id}`,
						value: inactive,
						label: `${row.name}: закрытых`,
						color: inactiveColor(row.id),
					},
				]
			}),
		[rows],
	)

	const empty = buckets.length === 0

	return (
		<Paper sx={{ p: 3, border: '1px solid #e5e7eb', borderRadius: '12px', height: '100%' }} elevation={0}>
			<Box
				sx={{
					display: 'flex',
					flexWrap: 'wrap',
					justifyContent: 'space-between',
					alignItems: { xs: 'flex-start', sm: 'center' },
					gap: 1,
					mb: 2,
				}}
			>
				<Typography variant='h6' sx={{ fontWeight: 600 }}>
					{title}
				</Typography>
				<TopNSelect value={topN} onChange={onTopNChange} />
			</Box>

			{empty ? (
				isLoading ? (
					<Skeleton variant='rounded' height={280} />
				) : (
					<Box sx={{ py: 6, textAlign: 'center', color: '#6b7280' }}>
						<Typography variant='body2'>Нет данных за выбранный период</Typography>
					</Box>
				)
			) : (
				<>
					<PieChart
						height={280}
						hideLegend
						onItemClick={handleItemClick}
						series={[
							{
								id: 'inner',
								data: innerSeries,
								innerRadius: '20%',
								outerRadius: '60%',
								paddingAngle: 1,
								cornerRadius: 8,
								arcLabel: 'value',
								arcLabelMinAngle: 10,
								highlightScope: { fade: 'global', highlight: 'item' },
							},
							{
								id: 'outer',
								data: outerSeries,
								innerRadius: '64%',
								outerRadius: '100%',
								paddingAngle: 1,
								cornerRadius: 8,
								arcLabel: 'value',
								arcLabelMinAngle: 10,
								highlightScope: { fade: 'global', highlight: 'item' },
							},
						]}
						margin={{ top: 4, bottom: 4, left: 4, right: 4 }}
					/>

					<Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1.5 }}>
						{rows.map(row => (
							<Box
								key={row.id}
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'center',
									gap: 0.5,
									px: 1,
									py: 0.5,
								}}
							>
								<Box
									sx={{
										width: 10,
										height: 10,
										borderRadius: '50%',
										bgcolor: activeColor(row.id),
									}}
								/>
								<Typography sx={{ color: 'text.secondary', fontSize: '0.875rem', lineHeight: 1 }}>
									{row.name} · {row.total}
								</Typography>
							</Box>
						))}
					</Box>

					<Typography sx={{ color: 'text.secondary', display: 'block', mt: 1, fontSize: '0.75rem', px: 1 }}>
						Внутреннее кольцо — всего заявок, внешнее — активные и закрытые
					</Typography>
				</>
			)}
		</Paper>
	)
}
