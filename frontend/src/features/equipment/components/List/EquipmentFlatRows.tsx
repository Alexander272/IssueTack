import type { FC } from 'react'

import type { IEquipment } from '../../types/equipment'
import { EquipmentRow } from '../Table'

type Props = {
	items: IEquipment[]
	onOpen: (id: string) => void
}

// Плоский серверный список (поиск или фильтр по типу) без иерархии: контейнер
// строки приходит с сервера (containerName), чтобы не терять подпись «внутри …».
export const EquipmentFlatRows: FC<Props> = ({ items, onOpen }) => (
	<>
		{items.map(item => (
			<EquipmentRow
				key={item.id}
				node={item}
				depth={0}
				parentName={item.containerName ?? undefined}
				showToggle={false}
				onOpen={onOpen}
			/>
		))}
	</>
)
