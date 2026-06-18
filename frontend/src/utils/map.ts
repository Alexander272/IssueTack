export function replacer(_key: unknown, value: Map<unknown, unknown>) {
	if (value instanceof Map) {
		return {
			dataType: 'Map',
			value: Array.from(value.entries()), // or with spread: value: [...value]
		}
	} else {
		return value
	}
}

export function reviver(
	_key: unknown,
	value: { dataType: string; value: Iterable<readonly [unknown, unknown]> | null | undefined } | null
) {
	if (typeof value === 'object' && value !== null) {
		if (value.dataType === 'Map') {
			return new Map(value.value)
		}
	}
	return value
}
