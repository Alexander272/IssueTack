// Список вложенных элементов внутри карточки-родителя: светло-серая подложка и
// вертикальная линия-связка слева (tree-view indicator). Один стиль на карточку
// единицы и групповую карточку-«партию», чтобы дерево читалось одинаково.
//
// Отдельный файл, а не EquipmentCard: правило react-refresh требует не смешивать
// экспорт компонента и константы (как в treeState.ts).
export const nestedListSx = {
	mt: 0.75,
	ml: 0.25,
	py: 0.5,
	pl: 1,
	pr: 0.5,
	borderLeft: '2px solid #e5e7eb',
	bgcolor: '#f9fafb',
	borderRadius: '0 8px 8px 0',
	display: 'flex',
	flexDirection: 'column',
	gap: 0.25,
} as const
