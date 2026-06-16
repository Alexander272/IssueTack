import { useEffect } from 'react'

import { useAppDispatch, useAppSelector } from '@/hooks/redux'
import { getToken, setUser } from '@/features/user/userSlice'
import { useRefreshQuery } from '../authApiSlice'

export function useRefresh() {
	const { data, isSuccess, isError, isFetching, isLoading } = useRefreshQuery(null)
	const dispatch = useAppDispatch()

	// Достаем пользователя напрямую из стора
	const token = useAppSelector(getToken)

	useEffect(() => {
		if (isSuccess && data) {
			dispatch(setUser(data.data))
		}
	}, [isSuccess, data, dispatch])

	// ЛОГИКА ГОТОВНОСТИ:
	// 1. Мы НЕ готовы, если запрос всё еще идет (isLoading или isFetching)
	// 2. Если запрос успешен, мы НЕ готовы, пока user в сторе всё еще пустой (ждем завершения dispatch)
	// 3. Если ошибка — мы готовы (покажем страницу логина)

	const isFinished = !isLoading && !isFetching
	const hasUserInStore = isSuccess && !!token

	const ready = isFinished && (isError || hasUserInStore)

	return { ready }
}
