// Фильтр по типу в тулбаре: всё / единицы / комплекты.
export type TypeFilter = 'all' | 'unit' | 'kit'

export type EquipmentFilters = {
	search: string
	typeFilter: TypeFilter
	status: string
}
