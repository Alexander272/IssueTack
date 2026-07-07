import type { Column } from 'react-datasheet-grid'

import { TextAreaComponent, type TextAreaOptions } from './TextAreaComponent'

// Экспортируем как функцию, возвращающую колонку для работы с типом string | null
export const textAreaColumn = (options: TextAreaOptions = {}): Column<string | null, TextAreaOptions> => ({
	component: TextAreaComponent,
	columnData: options,
	disabled: options.disabled,
	deleteValue: () => null,
	copyValue: ({ rowData }) => rowData,
	pasteValue: ({ value }) => value || null,
})
