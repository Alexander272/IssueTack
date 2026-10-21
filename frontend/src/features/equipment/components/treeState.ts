import { createContext, useContext } from 'react'

// Состояние списка, которое нужно строкам глубоко в рекурсии дерева: раскрытие,
// переключение узла и открытие карточки. Раньше его протаскивали пропсами через
// EquipmentRootRows → TreeNode → NodeChildren (три уровня одних и тех же значений).
// Контекст убирает протаскивание: строка получает только узел и глубину.
//
// Файл без компонентов (контекст и хук) — правило react-refresh требует не
// смешивать экспорт компонента и не-компонента. Провайдер ставит EquipmentList.
export type EquipmentTreeState = {
	// tree — режим-дерево: у строк рисуются шевроны раскрытия. В плоском списке false.
	tree: boolean
	// mobile — узкий экран: строки таблицы подменяются карточками (как TaskCardList
	// на странице заявок). Значение считается один раз в EquipmentList и живёт в
	// контексте, чтобы строки глубоко в рекурсии не подписывались каждая на media query.
	mobile: boolean
	expanded: Set<string>
	toggle: (id: string) => void
	open: (id: string) => void
	// search — текущая поисковая фраза (debounce-нутая). Дети узла запрашиваются тем
	// же фильтром, что и корни: иначе раскрытый состав показывал бы всё подряд, тогда
	// как корни уже отфильтрованы.
	search: string
}

// Дефолт работает и вне провайдера (плоский список, отдельная строка): дерево
// выключено, десктопная раскладка, действия — no-op.
export const EquipmentTreeContext = createContext<EquipmentTreeState>({
	tree: false,
	mobile: false,
	expanded: new Set(),
	toggle: () => {},
	open: () => {},
	search: '',
})

export const useEquipmentTree = () => useContext(EquipmentTreeContext)
