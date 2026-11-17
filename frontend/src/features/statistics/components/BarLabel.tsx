import { BarLabel, type BarLabelProps } from '@mui/x-charts/BarChart'

// Кастомный рендер подписи с адаптивным цветом текста
export function CustomBarLabel(props: BarLabelProps) {
	const { seriesId, width, ...other } = props

	// Не рисуем цифру, если ширина полосы меньше 18px (чтобы не налезала на края)
	if (typeof width === 'number' && width < 8) {
		return null
	}

	// Для светлых серий (закрытые задачи) делаем темный текст, для ярких — белый
	const isLightSeries = String(seriesId).startsWith('closed')

	return (
		<BarLabel
			{...other}
			seriesId={seriesId}
			width={width}
			style={{
				...props.style,
				fontSize: 11,
				fontWeight: 600,
				fill: isLightSeries ? '#2b3c57' : '#ffffff',
			}}
		/>
	)
}
