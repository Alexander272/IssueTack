import { useCallback, useState } from 'react'

/** Разделы/категории развёрнуты по умолчанию; храним явно свёрнутые. */
export const useCollapse = () => {
	const [closed, setClosed] = useState<Set<string>>(() => new Set())

	const toggle = useCallback((key: string) => {
		setClosed(prev => {
			const next = new Set(prev)
			if (next.has(key)) next.delete(key)
			else next.add(key)
			return next
		})
	}, [])

	const closeAll = useCallback((keys: string[]) => setClosed(new Set(keys)), [])
	const openAll = useCallback(() => setClosed(new Set()), [])

	return { closed, toggle, closeAll, openAll }
}
