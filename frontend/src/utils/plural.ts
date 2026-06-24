export function pluralize(count: number, forms: [string, string, string]): string {
	if (count === 1) return forms[0]
	if (count > 1 && count < 5) return forms[1]
	return forms[2]
}
