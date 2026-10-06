package constants

type ctxKey string

const (
	CtxUser ctxKey = "user_context"
	// CtxRealm — реалм, под которым CheckPermissions уже проверил членство
	// и права. Это авторитетное значение: именно оно ушло в Enforce, поэтому
	// брать realm из заголовка повторно в хендлере нельзя (middleware ещё и
	// поддерживает ?realm=, то есть источников значения два).
	CtxRealm ctxKey = "realm_context"
)
