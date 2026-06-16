export const HttpCodes = Object.freeze({
	OK: 200 as const,
	CREATED: 201 as const,
	BAD_REQUEST: 400 as const,
	UNAUTHORIZED: 401 as const,
	FORBIDDEN: 403 as const,
	NOT_FOUND: 404 as const,
	TO_LARGE: 413 as const,
	TO_MANY_REQUESTS: 429 as const,
})
