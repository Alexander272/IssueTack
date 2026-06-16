import { getPermissions } from '@/features/user/userSlice'
import { useAppSelector } from '@/hooks/redux'

// export const useCheckPermission = (rule: string) => {
// 	const permissions = useAppSelector(getPermissions)
// 	if (!permissions.length) return false

// 	for (let i = 0; i < permissions.length; i++) {
// 		if (permissions[i] === rule) return true
// 	}
// 	return false
// }

export const useCheckPermission = (requiredRules: string | string[]): boolean => {
	const permissions = useAppSelector(getPermissions)

	if (!permissions || !permissions.length) return false

	// Приводим к массиву, если пришла одиночная строка
	const rulesToCheck = Array.isArray(requiredRules) ? requiredRules : [requiredRules]

	// Если хотя бы одно из требуемых правил (rulesToCheck) совпадает с имеющимися (permissions)
	return rulesToCheck.some(requiredRule =>
		permissions.some(p => {
			if (p === requiredRule || p === '*' || p === '*:*') return true

			const [pObj, pAct] = p.split(':')
			const [reqObj, reqAct] = requiredRule.split(':')

			const objMatch = pObj === '*' || pObj === reqObj
			const actMatch = pAct === '*' || pAct === reqAct

			return objMatch && actMatch
		}),
	)
}
