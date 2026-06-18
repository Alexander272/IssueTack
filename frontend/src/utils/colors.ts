export const stringToColor = (string: string) => {
	let hash = 0
	for (let i = 0; i < string.length; i++) {
		hash = string.charCodeAt(i) + ((hash << 5) - hash)
	}
	let color = '#'
	for (let i = 0; i < 3; i++) {
		const value = (hash >> (i * 8)) & 0xff
		color += `00${value.toString(16)}`.slice(-2)
	}
	return color
}

export const stringToHSLA = (str: string) => {
	let hash = 0
	for (let i = 0; i < str.length; i++) {
		hash = str.charCodeAt(i) + ((hash << 5) - hash)
	}

	// 1. Hue (Цвет): от 0 до 360
	const h = Math.abs(hash) % 360

	// 2. Saturation (Насыщенность): 40-60% для мягкости
	const s = 50

	// 3. Lightness (Светлость): 90-95% для очень светлого фона
	const l = 92

	return {
		bg: `hsla(${h}, ${s}%, ${l}%, 1)`,
		border: `hsla(${h}, ${s}%, ${l - 20}%, 1)`, // Граница чуть темнее фона
		text: `hsla(${h}, ${s + 20}%, ${l - 60}%, 1)`, // Текст контрастный и темный
	}
}
