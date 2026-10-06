// Ограничение ширины чипа фильтра: если значений много, показываем столько,
// сколько помещается, а остаток отдаём суффиксом « +N».
export const CHIP_MAX_WIDTH = 300

// Шрифт должен совпадать с FilterChip (fontSize 0.875rem, fontWeight 500).
const CHIP_FONT = '500 14px Roboto, Helvetica, Arial, sans-serif'

// Паддинги чипа (12 + 12) и место под delete-иконку.
const CHIP_TEXT_BUDGET = CHIP_MAX_WIDTH - 52

let context: CanvasRenderingContext2D | null = null

const getContext = (): CanvasRenderingContext2D | null => {
	if (typeof document === 'undefined') return null
	if (!context) {
		context = document.createElement('canvas').getContext('2d')
		if (context) context.font = CHIP_FONT
	}
	return context
}

const measure = (text: string): number => {
	const ctx = getContext()
	// Грубая оценка, если canvas недоступен (SSR/тесты).
	return ctx ? ctx.measureText(text).width : text.length * 7.2
}

export interface FittedChipLabel {
	// «Префикс: v1, v2» — без суффикса скрытых значений.
	text: string
	// Сколько значений не поместилось.
	hidden: number
}

export const fitChipLabel = (prefix: string, values: string[]): FittedChipLabel => {
	if (values.length === 0) return { text: `${prefix}: `, hidden: 0 }

	for (let shown = values.length; shown >= 1; shown--) {
		const hidden = values.length - shown
		const suffix = hidden > 0 ? ` +${hidden}` : ''
		const text = `${prefix}: ${values.slice(0, shown).join(', ')}`
		if (measure(text + suffix) <= CHIP_TEXT_BUDGET) return { text, hidden }
	}

	// Не влезает даже одно имя с суффиксом: отдаём первое имя, длину подгонит
	// CSS-ellipsis, а « +N» FilterChip рисует отдельным необрезáемым элементом.
	return { text: `${prefix}: ${values[0]}`, hidden: values.length - 1 }
}