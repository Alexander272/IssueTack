export function getChartColorFromText(hex: string, targetSaturation = 75, targetLightness = 50) {
	// 1. Очищаем HEX и переводим в RGB
	let c = hex.replace('#', '')
	if (c.length === 3)
		c = c
			.split('')
			.map(x => x + x)
			.join('')

	const r = parseInt(c.substring(0, 2), 16) / 255
	const g = parseInt(c.substring(2, 4), 16) / 255
	const b = parseInt(c.substring(4, 6), 16) / 255

	// 2. Находим Тон (Hue) из RGB
	const max = Math.max(r, g, b),
		min = Math.min(r, g, b)
	let h = 0

	if (max !== min) {
		const d = max - min
		switch (max) {
			case r:
				h = (g - b) / d + (g < b ? 6 : 0)
				break
			case g:
				h = (b - r) / d + 2
				break
			case b:
				h = (r - g) / d + 4
				break
		}
		h = Math.round(h * 60)
	}

	// 3. Возвращаем строку HSL с фиксированной насыщенностью и светлотой
	return `hsl(${h}, ${targetSaturation}%, ${targetLightness}%)`
}
