import type { IEquipment } from '../../types/equipment'
import { pluralize } from '@/utils/plural'

// Узел списка: у дерева приходят серверные признаки раскрытия, плоский список —
// обычной единицей, поэтому они опциональны. Тип общий для строки таблицы и карточки.
export type EquipmentRowNode = IEquipment & { hasChildren?: boolean; childCount?: number }

export type EquipmentRowMeta = {
	holder: string
	location: string
	isInstalled: boolean
	childCount: number
	// subtitle — вторая строка строки/карточки: инв. № плюс контекст (где установлено,
	// сколько компонентов или площадка складской единицы).
	subtitle: string
}

// Смысловой вид строки: комплект / установленный / вложенный / обычный. Один источник
// решения для строки таблицы и карточки (EquipmentRow / EquipmentCard), чтобы ветвление
// на «цвет фона» не дублировалось руками в каждом компоненте.
export type EquipmentRowKind = 'kit' | 'installed' | 'nested' | 'plain'

export const equipmentRowKind = (node: EquipmentRowNode, isInstalled: boolean, depth: number): EquipmentRowKind => {
	if (node.isKit) return 'kit'
	if (isInstalled) return 'installed'
	if (depth > 0) return 'nested'
	return 'plain'
}

// Метрики строки, общие для десктопной EquipmentRow и мобильной EquipmentCard:
// держатель, место, признак установленности и подпись второй строки. Одна функция,
// чтобы две разметки не разъезжались в толковании полей.
export const equipmentRowMeta = (node: EquipmentRowNode): EquipmentRowMeta => {
	const childCount = node.childCount ?? 0
	const holder = node.holderFullName || node.holderName || ''
	const location = node.location || node.siteName || ''
	const isInstalled = Boolean(node.parentId)

	const parts = [`Инв. №${node.inventoryNumber ?? '—'}`]
	if (isInstalled) parts.push(`внутри ${node.containerName ?? 'родителя'}`)
	else if (childCount) parts.push(`${childCount} ${pluralize(childCount, ['компонент', 'компонента', 'компонентов'])}`)
	else if (node.status === 'in_stock' && location) parts.push(location)

	return { holder, location, isInstalled, childCount, subtitle: parts.join(' · ') }
}
