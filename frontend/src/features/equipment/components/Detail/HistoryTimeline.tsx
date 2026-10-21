import { type ElementType, type FC, useMemo, useState } from 'react'
import { Box, Stack, ToggleButton, ToggleButtonGroup, Typography } from '@mui/material'
import dayjs from 'dayjs'
import {
	AlertTriangleIcon,
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
	defective: { Icon: AlertTriangleIcon, color: '#ca8a04', group: 'status' },
}

// Строка выдачи и событие журнала — два вида одной операции: обе сущности
// пишутся в одной транзакции при выдаче и возврате. Лента обязана показать
// операцию один раз, и связь между ними явная: событие называет строку, которую
// открыло (assignmentId), и строку, которую закрыло (closedAssignmentId).
// Сопоставлять по времени нельзя: момент выдачи приходит из формы (её
// оформляют задним числом, вплоть до полуночи), а момент записи в журнал —
// другое, разные факты; близость моментов ничего не значит (см. миграцию
// equipment_events_assignment_link).
//
// Состав комплекта — другое дело: там и строка, и событие пишутся серверным
// временем в одной транзакции, расхождение — доли секунды, и связь по
// компоненту (componentId) однозначна. Поэтому для состава окно остаётся.
const COMPOSITION_WINDOW_MS = 5000

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

// Лента истории единицы: события журнала плюс легаси-записи выдач и состава.
// Показывается днём: у каждого дня липкий заголовок, у записи — кружок с
// иконкой по типу и строка «было → стало» для изменений статуса,
// перемещений и состава.
export const HistoryTimeline: FC<Props> = ({ unit }) => {
	const [group, setGroup] = useState<Group | 'all'>('all')

	const { data: assignmentsData } = useGetEquipmentAssignmentsQuery(unit.id)
	// Журнал берём с запасом: обрезанный журнал не ломает ленту (не хватающая
	// операция покажется записью строки выдачи), но лишние записи лучше не
	// терять.
	const { data: eventsData } = useGetEquipmentEventsQuery({ id: unit.id, limit: 500 })
	const { data: compositionData } = useGetEquipmentCompositionHistoryQuery(unit.id, { skip: !unit.isKit })

	const entries = useMemo<Entry[]>(() => {
		const events = (eventsData?.data ?? [])
		const rows = assignmentsData?.data ?? []

		// Явная связь операции и строк выдачи. Событие называет строку, которую
		// открыло, и строку, которую закрыло: по этим ссылкам лента и понимает,
		// что строка и событие — один и тот же факт, и рисует его один раз.
		const rowsById = new Map(rows.map(r => [r.id, r]))
		const openedBy = new Map<string, IEquipmentEvent>() // строка → событие, открывшее её
		const closedBy = new Map<string, IEquipmentEvent>() // строка → событие, закрывшее её
		events.forEach(e => {
			if (e.assignmentId) openedBy.set(e.assignmentId, e)
			if (e.closedAssignmentId) closedBy.set(e.closedAssignmentId, e)
		})

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

		// placeText — подпись места строкой (для примечания к записи).
		const placeText = (v: Record<string, unknown> | null) =>
			v ? placeLabel(str(v.siteName), str(v.location)) : ''

		// holderOf: снимок несёт и ФИО сотрудника, и свободный текст держателя.
		const holderOf = (v: Record<string, unknown> | null): Value | undefined => {
			if (!v) return undefined
			const label = str(v.userName) ?? str(v.holderName)
			return label ? { label, color: '#16a34a' } : undefined
		}

		// holderOfRow — держатель по строке выдачи (слепок ФИО резолвит бэкенд).
		const holderOfRow = (a: IEquipmentAssignment): Value | undefined => {
			const label = a.holderFullName || a.holderName
			return label ? { label, color: '#16a34a' } : undefined
		}

		// dueNote — срок возврата строки выдачи. У закрытой строки он уже
		// неактуален и только засорял бы ленту.
		const dueNote = (a?: IEquipmentAssignment) =>
			a?.returnDueAt && !a.unassignedAt
				? `${
						dayjs(a.returnDueAt).isBefore(dayjs(), 'day') ? 'Возврат просрочен с ' : 'Вернуть до '
					}${dayjs(a.returnDueAt).format('DD.MM.YYYY')}`
				: ''

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
			const eventComment = e.comment || undefined

			switch (e.eventType) {
				case 'assigned': {
					// Строка, которую открыло событие, — та же операция: она
					// дополняет запись журнала (место, срок возврата, комментарий),
					// но отдельной записью не рисуется.
					const row = e.assignmentId ? rowsById.get(e.assignmentId) : undefined
					const holder = holderOf(e.newValue) ?? (row ? holderOfRow(row) : undefined)
					// Выдал себе — не печатаем одно и то же имя дважды: актор и
					// получатель совпадают, получателя показывает чип.
					const actor = holder && holder.label === base.actor ? undefined : base.actor
					const place = placeText(e.newValue) || (row ? placeLabel(row.siteName, row.location) : '')
					const noteComment = row?.comment || eventComment
					return {
						...base,
						actor,
						action: 'Выдано',
						to: holder,
						note: [place, dueNote(row), noteComment].filter(Boolean).join(' · ') || undefined,
					}
				}
				case 'unassigned': {
					// Возврат закрывает прежнюю выдачу и открывает строку «на складе» —
					// обе строки названы событием явно.
					const opened = e.assignmentId ? rowsById.get(e.assignmentId) : undefined
					const closed = e.closedAssignmentId ? rowsById.get(e.closedAssignmentId) : undefined
					const place = placeText(e.newValue) || (opened ? placeLabel(opened.siteName, opened.location) : '')
					const noteComment = closed?.comment || opened?.comment || eventComment
					return {
						...base,
						action: 'Возвращено на склад',
						to: place ? { label: place, color: '#2563eb' } : undefined,
						note: [dueNote(opened), noteComment].filter(Boolean).join(' · ') || undefined,
					}
				}
				case 'moved':
					return {
						...base,
						action: 'Перемещено',
						from: placeOf(e.oldValue, '#94a3b8'),
						to: placeOf(e.newValue, '#2563eb'),
						note: eventComment,
					}
				case 'status_changed':
					return { ...base, action: 'Изменён статус', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: eventComment }
				case 'repaired':
					return { ...base, action: 'Отправлено в ремонт', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: eventComment }
				case 'written_off':
					return { ...base, action: 'Списано', from: statusOf(e.oldValue), to: statusOf(e.newValue), note: eventComment }
				case 'defective': {
					// Дефект — факт с данными: описание лежит в снимке события, а
					// дата обнаружения — отдельным чипом, потому что её часто
					// ставят раньше момента записи.
					const description = str(e.newValue?.description)
					const foundAt = str(e.newValue?.foundAt)
					return {
						...base,
						action: 'Обнаружен дефект',
						note: [description, eventComment].filter(Boolean).join(' · ') || undefined,
						to: foundAt
							? { label: dayjs(foundAt).locale('ru').format('D MMMM YYYY'), color: '#ca8a04' }
							: undefined,
					}
				}
				case 'component_added':
					return { ...base, action: 'Компонент добавлен в комплект', to: componentOf(e.newValue, e.componentId), note: eventComment }
				case 'component_removed':
					return { ...base, action: 'Компонент исключён из комплекта', from: componentOf(e.oldValue, e.componentId), note: eventComment }
				case 'component_replaced':
					return {
						...base,
						action: 'Компонент заменён',
						from: componentOf(e.oldValue, e.componentId),
						to: componentOf(e.newValue, e.componentId),
						note: eventComment,
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
						note: [eventComment, details].filter(Boolean).join(' · ') || undefined,
					}
				}
			}
		}

		// covered сообщает, что операцию уже показал журнал событий.
		const covered = (type: EquipmentEventType, date: string, componentId?: string | null) =>
			events.some(
				e =>
					e.eventType === type &&
					Math.abs(time(e.createdAt) - time(date)) < COMPOSITION_WINDOW_MS &&
					(!componentId || !e.componentId || e.componentId === componentId),
			)

		const fromAssignment = (a: IEquipmentAssignment): Entry[] => {
			const list: Entry[] = []
			// Начало периода строки журнал уже показал (событие открыло её) —
			// отдельной записью не рисуем. Сюда попадают только выдачи, сделанные
			// до появления журнала: их операций в журнале нет вовсе.
			if (openedBy.has(a.id)) return list

			const holder = a.holderFullName || a.holderName
			const place = placeLabel(a.siteName, a.location)
			const due = dueNote(a)
			if (holder) {
				// Выдал себе — не печатаем одно и то же имя дважды.
				const actor = a.assignedByName === holder ? undefined : a.assignedByName || undefined
				list.push({
					id: `assign-${a.id}`,
					date: a.assignedAt,
					Icon: UserPlusIcon,
					color: '#16a34a',
					group: 'transfer',
					actor,
					action: 'Выдано',
					to: { label: holder, color: '#16a34a' },
					note: [place, due, a.comment].filter(Boolean).join(' · ') || undefined,
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
			// Конец периода: если закрытие уже показал журнал (возврат или
			// переоформление выдачи), отдельную запись «Возвращено на склад» рисуем
			// только когда закрытия в журнале нет вовсе.
			if (a.unassignedAt && !closedBy.has(a.id)) {
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
			...(rows ?? []).flatMap(fromAssignment),
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
