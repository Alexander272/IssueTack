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
}

// Дефолт работает и вне провайдера (плоский список, отдельная строка): дерево
// выключено, десктопная раскладка, действия — no-op.
export const EquipmentTreeContext = createContext<EquipmentTreeState>({
	tree: false,
	mobile: false,
	expanded: new Set(),
	toggle: () => {},
	open: () => {},
})

export const useEquipmentTree = () => useContext(EquipmentTreeContext)
