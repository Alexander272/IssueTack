import { type ElementType, type FC, useMemo, useState } from 'react'
import { Box, Stack, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material'
import dayjs from 'dayjs'
import {
	ArchiveRestoreIcon,
	ArrowRightIcon,
	ClockIcon,
	MapPinIcon,
	PackageMinusIcon,
	PencilIcon,
	PlusIcon,
	RefreshCwIcon,
	Trash2Icon,
	Undo2Icon,
	UserPlusIcon,
	WrenchIcon,
} from 'lucide-mui'

import type {
	EquipmentEventType,
	EquipmentStatus,
	IEquipment,
	IEquipmentAssignment,
	IEquipmentEvent,
	IKitComposition,
} from '../../types/equipment'
import { getEquipmentEventLabel, getEquipmentStatus } from '../../types/equipment'
import {
	useGetEquipmentAssignmentsQuery,
	useGetEquipmentCompositionHistoryQuery,
	useGetEquipmentEventsQuery,
} from '../../equipmentApiSlice'

type Group = 'composition' | 'status' | 'transfer'

// Value — одно из значений строки «было → стало»: подпись и цвет чипа.
type Value = { label: string; color: string }

type Entry = {
	id: string
	date: string
	Icon: ElementType
	color: string
	group: Group
	// actor — «Фамилия Имя» того, кто совершил действие: пусто, если неизвестен.
	actor?: string
	action: string
	from?: Value
	to?: Value
	note?: string
}

const GROUPS: { value: Group | 'all'; label: string }[] = [
	{ value: 'all', label: 'Все события' },
	{ value: 'composition', label: 'Состав' },
	{ value: 'status', label: 'Статус' },
	{ value: 'transfer', label: 'Перемещения' },
]

const EVENT_META: Record<EquipmentEventType, { Icon: ElementType; color: string; group: Group }> = {
	created: { Icon: PlusIcon, color: '#2563eb', group: 'status' },
	updated: { Icon: PencilIcon, color: '#6b7280', group: 'status' },
	assigned: { Icon: UserPlusIcon, color: '#16a34a', group: 'transfer' },
	unassigned: { Icon: ArchiveRestoreIcon, color: '#2563eb', group: 'transfer' },
	moved: { Icon: MapPinIcon, color: '#2563eb', group: 'transfer' },
	status_changed: { Icon: RefreshCwIcon, color: '#ea580c', group: 'status' },
	component_added: { Icon: PlusIcon, color: '#7c3aed', group: 'composition' },
	component_removed: { Icon: PackageMinusIcon, color: '#dc2626', group: 'composition' },
	component_replaced: { Icon: Undo2Icon, color: '#7c3aed', group: 'composition' },
	repaired: { Icon: WrenchIcon, color: '#ea580c', group: 'status' },
	written_off: { Icon: Trash2Icon, color: '#dc2626', group: 'status' },
}

// Легаси-записи (выдачи и состав) дублируют события: обе сущности пишутся в одной
// транзакции. Запись справочника показываем, только если журнал её не покрыл —
// иначе в ленте будет по два одинаковых пункта на каждую операцию.
const EVENT_WINDOW_MS = 5000

const time = (d: string) => dayjs(d).valueOf()

// str достаёт непустую строку из JSON-значения события.
const str = (v: unknown) => (typeof v === 'string' && v ? v : null)

const fmtTime = (d: string) => dayjs(d).format('HH:mm')

// Ключ дня — начало суток в миллисекундах: сравнение идёт по точкам времени, а не
// по строке, иначе разбор 'YYYY-MM-DD' как UTC сдвигал бы день в минусовых зонах.
const dayKey = (d: string) => dayjs(d).startOf('day').valueOf()

const dayLabel = (key: number) => {
	if (dayjs(key).isSame(dayjs(), 'day')) return 'Сегодня'
	if (dayjs(key).isSame(dayjs().subtract(1, 'day'), 'day')) return 'Вчера'
	return dayjs(key).locale('ru').format('D MMMM YYYY')
}

type Props = {
	unit: IEquipment
}

// Лента истории единицы: события журнала плюс легаси-записи выдач и состава (см.
// EVENT_WINDOW_MS). Показывается днём: у каждого дня липкий заголовок, у записи —
// кружок с иконкой по типу и строка «было → стало» для изменений статуса,
// перемещений и состава.
export const HistoryTimeline: FC<Props> = ({ unit }) => {
	const [group, setGroup] = useState<Group | 'all'>('all')

	const { data: assignmentsData } = useGetEquipmentAssignmentsQuery(unit.id)
	const { data: eventsData } = useGetEquipmentEventsQuery({ id: unit.id, limit: 100 })
	const { data: compositionData } = useGetEquipmentCompositionHistoryQuery(unit.id, { skip: !unit.isKit })

	const entries = useMemo<Entry[]>(() => {
		const events = (eventsData?.data ?? [])

		// placeLabel собирает подпись места из пары «площадка + локация».
		const placeLabel = (siteName: string | null, location: string | null) =>
			[siteName, location].filter(Boolean).join(', ')

		// placeOf читает снимок события: название площадки лежит в JSON рядом с id,
		// поэтому справочник площадок клиенту не нужен.
		const placeOf = (v: Record<string, unknown> | null, color: string): Value | undefined => {
			if (!v) return undefined
			const label = placeLabel(str(v.siteName), str(v.location))
			return label ? { label, color } : undefined
		}

		// holderOf: снимок несёт и ФИО сотрудника, и свободный текст держателя.
		const holderOf = (v: Record<string, unknown> | null): Value | undefined => {
			if (!v) return undefined
			const label = str(v.userName) ?? str(v.holderName)
			return label ? { label, color: '#16a34a' } : undefined
		}

		const statusOf = (v: Record<string, unknown> | null): Value | undefined => {
			const status = v ? str(v.status) : null
			if (!status) return undefined
			const info = getEquipmentStatus(status as EquipmentStatus)
			return { label: info.label, color: info.color }
		}

		const componentOf = (v: Record<string, unknown> | null, componentId: string | null): Value | undefined => {
			const name = v ? str(v.name) : null
			if (name) return { label: name, color: '#7c3aed' }
			const id = (v ? str(v.equipmentId) : null) ?? componentId
			if (!id) return undefined
			return { label: `#${id.slice(0, 8)}`, color: '#7c3aed' }
		}

		// Записи журнала.
		const fromEvent = (e: IEquipmentEvent): Entry | null => {
			const meta = EVENT_META[e.eventType]
			if (!meta) return null
			const base = {
				id: `event-${e.id}`,
				date: e.createdAt,
				Icon: meta.Icon,
				color: meta.color,
				group: meta.group,
				actor: e.userName || undefined,
			}
			const comment = e.comment || undefined

			switch (e.eventType) {
				case 'assigned': {
					const holder = holderOf(e.newValue)
					// Выдал себе — не печатаем одно и то же имя дважды: актор и
					// получатель совпадают, получателя показывает чип.
					const actor = holder && holder.label === base.actor ? undefined : base.actor
					return { ...base, actor, action: 'Выдано', to: holder, note: comment }
				}
				case 'unassigned':
					return { ...base, action: 'Возвращено на склад', to: placeOf(e.newValue, '#2563eb'), note: comment }
				case 'moved':
					return {
						...base,
						action: 'Перемещено',
						from: placeOf(e.oldValue, '#94a3b8'),
						to: placeOf(e.newValue, '#2563eb'),
						note: comment,
					}
				case 'status_changed':
					return { ...base, action: 'Изменён статус', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: comment }
				case 'repaired':
					return { ...base, action: 'Отправлено в ремонт', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: comment }
				case 'written_off':
					return { ...base, action: 'Списано', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: comment }
				case 'component_added':
					return { ...base, action: 'Компонент добавлен в комплект', to: componentOf(e.newValue, e.componentId), note: comment }
				case 'component_removed':
					return { ...base, action: 'Компонент исключён из комплекта', from: componentOf(e.oldValue, e.componentId), note: comment }
				case 'component_replaced':
					return {
						...base,
						action: 'Компонент заменён',
						from: componentOf(e.oldValue, e.componentId),
						to: componentOf(e.newValue, e.componentId),
						note: comment,
					}
				default: {
					// «Создано»: опознавательные поля снимка — в примечании, чтобы не
					// повторять имя единицы из шапки карточки.
					const details = [
						str(e.newValue?.serialNumber) && `S/N ${str(e.newValue?.serialNumber)}`,
						str(e.newValue?.inventoryNumber) && `Инв. № ${str(e.newValue?.inventoryNumber)}`,
					]
						.filter(Boolean)
						.join(' · ')
					return {
						...base,
						action: e.eventType === 'created' ? 'Добавлено оборудование' : getEquipmentEventLabel(e.eventType),
						note: [comment, details].filter(Boolean).join(' · ') || undefined,
					}
				}
			}
		}

		// covered сообщает, что операцию уже показал журнал событий.
		const covered = (type: EquipmentEventType, date: string, componentId?: string | null) =>
			events.some(
				e =>
					e.eventType === type &&
					Math.abs(time(e.createdAt) - time(date)) < EVENT_WINDOW_MS &&
					(!componentId || !e.componentId || e.componentId === componentId),
			)

		// transferEventAt — событие журнала, покрывающее строку справочника выдач:
		// и выдача, и возврат. Тип события говорит, чем была операция.
		const transferEventAt = (date: string) =>
			events.find(
				e =>
					(e.eventType === 'assigned' || e.eventType === 'unassigned') &&
					Math.abs(time(e.createdAt) - time(date)) < EVENT_WINDOW_MS,
			)

		const fromAssignment = (a: IEquipmentAssignment): Entry[] => {
			const list: Entry[] = []
			// Строка справочника дублирует событие журнала: обе сущности пишутся
			// одной транзакцией и делят метку времени, поэтому журнал её покрывает и
			// отдельный пункт не нужен. Вид операции берём у события: строка возврата
			// (держателя нет) по полям не отличается от выдачи в одну площадку.
			if (transferEventAt(a.assignedAt)) return list

			const holder = a.holderFullName || a.holderName
			const place = placeLabel(a.siteName, a.location)
			// Срок возврата показываем у открытой выдачи (возврата ещё не было):
			// у закрытой строки он уже неактуален и только засорял бы ленту.
			const dueNote =
				a.returnDueAt && !a.unassignedAt
					? `${
							dayjs(a.returnDueAt).isBefore(dayjs(), 'day') ? 'Возврат просрочен с ' : 'Вернуть до '
						}${dayjs(a.returnDueAt).format('DD.MM.YYYY')}`
					: null
			if (holder) {
				list.push({
					id: `assign-${a.id}`,
					date: a.assignedAt,
					Icon: UserPlusIcon,
					color: '#16a34a',
					group: 'transfer',
					actor: a.assignedByName || undefined,
					action: 'Выдано',
					to: { label: holder, color: '#16a34a' },
					note: [place, dueNote, a.comment].filter(Boolean).join(' · ') || undefined,
				})
			} else if (place) {
				// Строка без держателя — единица стоит на площадке: это либо возврат
				// на склад, либо размещение без закрепления. Вид операции неизвестен,
				// поэтому нейтральное «Размещено».
				list.push({
					id: `place-${a.id}`,
					date: a.assignedAt,
					Icon: ArchiveRestoreIcon,
					color: '#2563eb',
					group: 'transfer',
					actor: a.assignedByName || undefined,
					action: 'Размещено',
					to: { label: place, color: '#2563eb' },
					note: a.comment || undefined,
				})
			}
			if (a.unassignedAt && !covered('unassigned', a.unassignedAt)) {
				list.push({
					id: `unassign-${a.id}`,
					date: a.unassignedAt,
					Icon: ArchiveRestoreIcon,
					color: '#2563eb',
					group: 'transfer',
					action: 'Возвращено на склад',
					note: a.comment || undefined,
				})
			}
			return list
		}

		const fromComposition = (c: IKitComposition): Entry[] => {
			const list: Entry[] = []
			const component = c.componentName || `#${c.componentId.slice(0, 8)}`
			if (!covered('component_added', c.startedAt, c.componentId)) {
				list.push({
					id: `comp-in-${c.id}`,
					date: c.startedAt,
					Icon: PlusIcon,
					color: '#7c3aed',
					group: 'composition',
					actor: c.startedByName || undefined,
					action: 'Компонент добавлен в комплект',
					to: { label: component, color: '#7c3aed' },
					note: c.reason || undefined,
				})
			}
			if (c.endedAt && !covered('component_removed', c.endedAt, c.componentId)) {
				list.push({
					id: `comp-out-${c.id}`,
					date: c.endedAt,
					Icon: PackageMinusIcon,
					color: '#dc2626',
					group: 'composition',
					actor: c.endedByName || undefined,
					action: 'Компонент исключён из комплекта',
					from: { label: component, color: '#dc2626' },
					note: c.reason || undefined,
				})
			}
			return list
		}

		const list: Entry[] = [
			...events.map(fromEvent).filter((e): e is Entry => e !== null),
			...(assignmentsData?.data ?? []).flatMap(fromAssignment),
			...(compositionData?.data ?? []).flatMap(fromComposition),
		]

		return list
			.filter(e => group === 'all' || e.group === group)
			.sort((a, b) => time(b.date) - time(a.date))
	}, [assignmentsData, eventsData, compositionData, group])

	// Группировка по дням: порядок дней — как в отсортированном списке (свежие выше).
	const days = useMemo(() => {
		const map = new Map<number, Entry[]>()
		entries.forEach(e => {
			const key = dayKey(e.date)
			const bucket = map.get(key)
			if (bucket) bucket.push(e)
			else map.set(key, [e])
		})
		return [...map.entries()]
	}, [entries])

	return (
		<Stack spacing={2}>
			<ToggleButtonGroup
				size='small'
				exclusive
				value={group}
				onChange={(_e, value: Group | 'all' | null) => value && setGroup(value)}
				sx={{
					alignSelf: 'flex-start',
					bgcolor: '#f1f5f9',
					borderRadius: '12px',
					p: 0.5,
					gap: 0.5,
					'& .MuiToggleButton-root': {
						border: 0,
						borderRadius: '9px',
						textTransform: 'none',
						fontSize: 12,
						fontWeight: 600,
						color: '#64748b',
						px: 1.5,
						py: 0.5,
						'&:hover': { bgcolor: '#e2e8f0' },
						'&.Mui-selected': {
							bgcolor: '#fff',
							color: '#1f2937',
							boxShadow: '0 1px 2px rgba(16, 24, 40, 0.08)',
							'&:hover': { bgcolor: '#fff' },
						},
					},
				}}
			>
				{GROUPS.map(g => (
					<ToggleButton key={g.value} value={g.value}>
						{g.label}
					</ToggleButton>
				))}
			</ToggleButtonGroup>

			{entries.length === 0 ? (
				<Box sx={{ py: 8, textAlign: 'center', color: '#9ca3af' }}>
					<ClockIcon sx={{ fontSize: 40, mb: 1, opacity: 0.5 }} />
					<Typography variant='body2'>Событий пока нет</Typography>
				</Box>
			) : (
				<Box>
					{days.map(([key, dayEntries]) => (
						<Box key={key}>
							{/* Липкий заголовок дня: при прокрутке видно, к какому дню
							    относится очередная запись. */}
							<Box
								sx={{
									position: 'sticky',
									top: 0,
									zIndex: 2,
									bgcolor: '#fff',
									py: 1,
									display: 'flex',
									alignItems: 'center',
									gap: 1.5,
								}}
							>
								<Typography sx={{ fontSize: 12, fontWeight: 700, color: '#64748b', whiteSpace: 'nowrap' }}>
									{dayLabel(key)}
								</Typography>
								<Box sx={{ flex: 1, height: '1px', bgcolor: '#eef2f6' }} />
							</Box>
							<Stack>
								{dayEntries.map((e, i) => (
									<TimelineRow key={e.id} entry={e} last={i === dayEntries.length - 1} />
								))}
							</Stack>
						</Box>
					))}
				</Box>
			)}
		</Stack>
	)
}

const TimelineRow: FC<{ entry: Entry; last: boolean }> = ({ entry, last }) => (
	<Box sx={{ display: 'flex', gap: 1.5 }}>
		<Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', width: 32, flexShrink: 0 }}>
			<Box
				sx={{
					width: 32,
					height: 32,
					borderRadius: '50%',
					bgcolor: `${entry.color}1a`,
					color: entry.color,
					display: 'flex',
					alignItems: 'center',
					justifyContent: 'center',
					flexShrink: 0,
				}}
			>
				<entry.Icon sx={{ fontSize: 15 }} />
			</Box>
			{!last && <Box sx={{ flex: 1, width: '1px', bgcolor: '#e5e7eb', my: 0.5 }} />}
		</Box>

		<Box sx={{ flex: 1, minWidth: 0, pb: last ? 1.5 : 2.5 }}>
			<Box sx={{ display: 'flex', alignItems: 'baseline', justifyContent: 'space-between', gap: 1 }}>
				<Typography sx={{ fontSize: 14, color: '#374151' }}>
					{entry.actor && <Box component='span' sx={{ fontWeight: 600, color: '#1f2937' }}>{entry.actor} </Box>}
					{entry.action}
				</Typography>
				<Typography sx={{ fontSize: 12, color: '#9ca3af', whiteSpace: 'nowrap' }}>{fmtTime(entry.date)}</Typography>
			</Box>

			{(entry.from || entry.to) && (
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mt: 0.75, flexWrap: 'wrap' }}>
					{entry.from && <ValueChip value={entry.from} strike />}
					{entry.from && entry.to && <ArrowRightIcon sx={{ fontSize: 14, color: '#9ca3af' }} />}
					{entry.to && <ValueChip value={entry.to} />}
				</Box>
			)}

			{entry.note && <Typography sx={{ fontSize: 12, color: '#6b7280', mt: 0.5 }}>{entry.note}</Typography>}
		</Box>
	</Box>
)

const ValueChip: FC<{ value: Value; strike?: boolean }> = ({ value, strike }) => (
	<Box
		sx={{
			display: 'inline-flex',
			alignItems: 'center',
			px: 1,
			py: 0.25,
			borderRadius: '999px',
			fontSize: 12,
			fontWeight: 500,
			bgcolor: `${value.color}1a`,
			color: value.color,
			border: '1px solid',
			borderColor: `${value.color}33`,
			textDecoration: strike ? 'line-through' : 'none',
		}}
	>
		{value.label}
	</Box>
)
