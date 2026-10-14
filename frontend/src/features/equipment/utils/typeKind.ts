import type { EquipmentTypeKind } from '../types/equipment'

// Акцентный цвет единицы по категории типа. Комплект (is_kit) всегда получает
// отдельный цвет — он вне перечисления kind.
export const KIT_COLOR = '#7c3aed'

const KIND_COLORS: Record<EquipmentTypeKind, string> = {
	device: '#0432a5',
	component: '#ea580c',
	periphery: '#6b7280',
	consumable: '#7c3aed',
}

export const equipmentAccent = (unit: { typeKind?: EquipmentTypeKind; isKit?: boolean }): string => {
	if (unit.isKit) return KIT_COLOR
	if (unit.typeKind) return KIND_COLORS[unit.typeKind]
	return '#6b7280'
}
