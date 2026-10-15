import type { FC } from 'react'

import type { IEquipment } from '../../types/equipment'
import { EquipmentRow } from '../Table'

type Props = {
	items: IEquipment[]
}

// Плоский серверный список (поиск или фильтр по типу) без иерархии: контейнер
// строки приходит с сервера (containerName), чтобы не терять подпись «внутри …».
export const EquipmentFlatRows: FC<Props> = ({ items }) => (
	<>
		{items.map(item => (
			<EquipmentRow key={item.id} node={item} />
		))}
	</>
)
