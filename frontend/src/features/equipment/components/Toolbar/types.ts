// Фильтр по типу в тулбаре: всё / единицы / комплекты.
export type TypeFilter = 'all' | 'unit' | 'kit'

export type EquipmentFilters = {
	search: string
	typeFilter: TypeFilter
	status: string
	// Место: площадка целиком (siteId) или локация внутри неё (оба поля). Пустые
	// значения означают «не фильтровать» — в запрос они не уходят.
	siteId: string | null
	locationId: string | null
}

// Режим-дерево доступен без поиска и без фильтра по месту: и поиск, и место дают
// плоский серверный список, потому что нужная единица может быть на любом уровне
// вложенности. Тип (единицы/комплекты) дерево не выключает — он сужает набор
// корней (isKit у запроса дерева).
export const isTreeMode = (query: string, hasPlace = false) => query.length === 0 && !hasPlace
