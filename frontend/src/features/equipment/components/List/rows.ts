import type { EquipmentTypeKind, IEquipmentTreeNode } from '../../types/equipment'
import type { TypeFilter } from '../Toolbar/types'

// ——— построение строк списка ———
// Чистые функции живут рядом с единственным потребителем (EquipmentList), а не в
// общем utils: сортировка узлов, группировка «партий» и раскрытие — его забота.

type Sortable = { isKit: boolean; typeKind: EquipmentTypeKind; name: string }

// Порядок узлов как в макете: сначала комплекты, затем единицы, затем комплектующие.
const kindOrder = (n: Sortable) => (n.isKit ? 0 : n.typeKind === 'device' ? 1 : 2)

export const sortNodes = <T extends Sortable>(nodes: T[]): T[] =>
	[...nodes].sort((a, b) => kindOrder(a) - kindOrder(b) || (a.name || '').localeCompare(b.name || '', 'ru'))

// Внутри раскрытой «партии» единицы идут по инвентарному номеру.
export const sortByInventory = (nodes: IEquipmentTreeNode[]) =>
	[...nodes].sort((a, b) => (a.inventoryNumber ?? '').localeCompare(b.inventoryNumber ?? ''))

// Складские единицы без детей группируются в «партию» по типу/названию/модели —
// как групповые строки на странице заявок. Узел, у которого есть дети, не
// группируется: он раскрывается как дерево.
const isGroupable = (n: IEquipmentTreeNode) => !n.isKit && n.status === 'in_stock' && !n.hasChildren
const groupKeyOf = (n: IEquipmentTreeNode) => `${n.typeKind}|${n.name}|${n.model}|${n.typeName}`

// Строка корневого уровня: одиночный узел либо «партия» одинаковых складских единиц.
export type RootBundle =
	| { type: 'single'; node: IEquipmentTreeNode }
	| { type: 'group'; key: string; items: IEquipmentTreeNode[] }

// Сворачивает корни страницы в строки: партия из одной единицы остаётся обычной строкой.
export const bundleRoots = (roots: IEquipmentTreeNode[]): RootBundle[] => {
	const bundles: RootBundle[] = []
	const index = new Map<string, number>()

	roots.forEach(n => {
		if (!isGroupable(n)) {
			bundles.push({ type: 'single', node: n })
			return
		}
		const key = groupKeyOf(n)
		const at = index.get(key)
		if (at === undefined) {
			index.set(key, bundles.length)
			bundles.push({ type: 'group', key, items: [n] })
			return
		}
		const bundle = bundles[at]
		if (bundle.type === 'group') bundle.items.push(n)
	})

	return bundles.map(b => (b.type === 'group' && b.items.length === 1 ? { type: 'single', node: b.items[0] } : b))
}

// Режим-дерево доступен без поиска и фильтра по типу.
export const isTreeMode = (query: string, typeFilter: TypeFilter) => query.length === 0 && typeFilter === 'all'

// Иммутабельный тумблер элемента множества (раскрытие узлов и «партий»).
export const toggleInSet = <T>(set: Set<T>, value: T): Set<T> => {
	const next = new Set(set)
	if (next.has(value)) next.delete(value)
	else next.add(value)
	return next
}
