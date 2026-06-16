import { type ContextMenuItem, createContextMenuComponent } from 'react-datasheet-grid'

export const ContextMenu = createContextMenuComponent((item: ContextMenuItem) => {
	switch (item.type) {
		case 'COPY':
			return <>Копировать</>
		case 'CUT':
			return <>Вырезать</>
		case 'PASTE':
			return <>Вставить</>
		case 'DELETE_ROW':
			return <>Удалить строку</>
		case 'DELETE_ROWS':
			return (
				<>
					Удалить строки с <b>{item.fromRow}</b> по <b>{item.toRow}</b>
				</>
			)
		case 'DUPLICATE_ROW':
			return <>Дублировать строку</>
		case 'DUPLICATE_ROWS':
			return (
				<>
					Дублировать строки с <b>{item.fromRow}</b> по <b>{item.toRow}</b>
				</>
			)
		case 'INSERT_ROW_BELLOW':
			return <>Вставить строку ниже</>
	}
})
