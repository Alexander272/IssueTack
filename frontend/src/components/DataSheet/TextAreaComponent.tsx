import { useLayoutEffect, useRef, memo } from 'react'
import type { CellProps } from 'react-datasheet-grid'

export type TextAreaOptions = {
	placeholder?: string
	disabled?: boolean
}

export const TextAreaComponent = memo(
	({
		active,
		rowData, // Важно: keyColumn передаст сюда именно значение поля (string | null), а не весь GridRow!
		setRowData, // Функция обновит конкретное поле ячейки
		focus,
		columnData,
	}: CellProps<string | null, TextAreaOptions>) => {
		const ref = useRef<HTMLTextAreaElement>(null)

		useLayoutEffect(() => {
			if (focus) {
				ref.current?.focus()
				const length = ref.current?.value.length || 0
				ref.current?.setSelectionRange(length, length)
			} else {
				ref.current?.blur()
			}
		}, [focus])

		return (
			<textarea
				ref={ref}
				disabled={columnData?.disabled || !active}
				placeholder={active ? columnData?.placeholder : undefined}
				value={rowData ?? ''}
				onChange={e => setRowData(e.target.value || null)}
				style={{
					width: '100%',
					height: '100%',
					border: 'none',
					outline: 'none',
					background: 'transparent',
					padding: '6px 8px',
					fontFamily: 'inherit',
					fontSize: 'inherit',
					resize: 'none',
					boxSizing: 'border-box',
				}}
				onKeyDown={e => {
					// Позволяем переносить строки по Enter, не перескакивая на следующую строку таблицы
					if (e.key === 'Enter') {
						e.stopPropagation()
					}
				}}
			/>
		)
	},
)
