import { useCallback, useRef } from 'react'

export const useDebounceFunc = <T>(callback: (...args: unknown[]) => T, delay: number) => {
	const timer = useRef<NodeJS.Timeout | undefined>(undefined)

	const debouncedCallback = useCallback(
		(...args: unknown[]) => {
			if (timer.current) {
				clearTimeout(timer.current)
			}
			timer.current = setTimeout(() => callback(...args), delay)
		},
		[callback, delay]
	)

	return debouncedCallback
}
