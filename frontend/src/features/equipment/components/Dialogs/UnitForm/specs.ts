// Характеристики единицы (specs) в форме правятся списком пар «название —
// значение». В самом specs лежит плоский словарь строк, поэтому редактируемая
// форма — массив пар, а не объект: у объекта нельзя завести вторую пустую строку
// (ключи совпадут), а на вводе ключ всегда промежуточно пустой.
//
// Вынесено из SpecsEditor.tsx: правило react-refresh требует, чтобы файл с
// компонентом экспортировал только компоненты.

export interface ISpecPair {
	key: string
	value: string
}

// specsToPairs разворачивает specs в редактируемый список. Порядок ключей — как
// отдал сервер; для формы этого достаточно.
export const specsToPairs = (specs?: Record<string, string> | null): ISpecPair[] =>
	Object.entries(specs ?? {}).map(([key, value]) => ({ key, value: String(value ?? '') }))

// pairsToSpecs собирает пары обратно в объект, отбрасывая полностью пустые строки
// и подрезая пробелы: пустая пара — это «строка добавлена, но не заполнена», а не
// характеристика без названия.
export const pairsToSpecs = (pairs: ISpecPair[]): Record<string, string> => {
	const specs: Record<string, string> = {}
	for (const { key, value } of pairs) {
		const name = key.trim()
		if (!name) continue
		specs[name] = value.trim()
	}
	return specs
}
