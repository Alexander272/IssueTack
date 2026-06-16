import React, { useLayoutEffect, useRef } from 'react'
import type { CellProps, Column } from 'react-datasheet-grid'

type TextAreaOptions = {
	placeholder?: string
	disabled?: boolean
}

const TextAreaComponent = React.memo(
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

// Экспортируем как функцию, возвращающую колонку для работы с типом string | null
export const textAreaColumn = (options: TextAreaOptions = {}): Column<string | null, TextAreaOptions> => ({
	component: TextAreaComponent,
	columnData: options,
	disabled: options.disabled,
	deleteValue: () => null,
	copyValue: ({ rowData }) => rowData,
	pasteValue: ({ value }) => value || null,
})
