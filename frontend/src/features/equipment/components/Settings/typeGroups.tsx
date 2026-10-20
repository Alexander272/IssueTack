import type { EquipmentTypeKind } from '../../types/equipment'

// Группа типов — это категория типа; порядок и подписи задают разбивку секции.
export type GroupKey = EquipmentTypeKind

export const GROUP_ORDER: GroupKey[] = ['device', 'component', 'periphery', 'consumable']

export const GROUP_LABELS: Record<GroupKey, string> = {
	device: 'Устройства',
	component: 'Комплектующие',
	periphery: 'Периферия',
	consumable: 'Расходники',
}

// Цвета бейджа категории в строке типа.
export const BADGE_STYLES: Record<GroupKey, { bgcolor: string; color: string }> = {
	device: { bgcolor: '#f3f4f6', color: '#374151' },
	component: { bgcolor: '#fff7ed', color: '#9a3412' },
	periphery: { bgcolor: '#eff6ff', color: '#1e40af' },
	consumable: { bgcolor: '#ecfdf5', color: '#065f46' },
}
