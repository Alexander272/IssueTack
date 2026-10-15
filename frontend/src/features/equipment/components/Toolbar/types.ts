// Фильтр по типу в тулбаре: всё / единицы / комплекты.
export type TypeFilter = 'all' | 'unit' | 'kit'

export type EquipmentFilters = {
	search: string
	typeFilter: TypeFilter
	status: string
}

// Режим-дерево доступен без поиска: поиск даёт плоский серверный список, потому что
// совпадение может оказаться на любом уровне вложенности. Тип (единицы/комплекты) дерево
// не выключает — он сужает набор корней (isKit у запроса дерева).
export const isTreeMode = (query: string) => query.length === 0
