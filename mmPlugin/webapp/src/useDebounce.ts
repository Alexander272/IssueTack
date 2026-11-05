import { useEffect, useState } from 'react'

// useDebounce отдаёт значение только после паузы: поиск в архиве не должен
// ходить на сервер на каждое нажатие клавиши. Сам инпут остаётся мгновенным.
export function useDebounce<T>(value: T, delay: number): T {
    const [debounced, setDebounced] = useState(value)

    useEffect(() => {
        const timer = setTimeout(() => setDebounced(value), delay)
        return () => clearTimeout(timer)
    }, [value, delay])

    return debounced
}
