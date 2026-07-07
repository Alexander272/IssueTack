import { useEffect } from 'react'

import { useAppDispatch, useAppSelector } from '@/hooks/redux'
import { getToken, setUser } from '@/features/user/userSlice'
import { getRealm, setRealm } from '@/features/realms/realmSlice'
import { useRefreshQuery } from '../authApiSlice'

export function useRefresh() {
	const { data, isSuccess, isError, isFetching, isLoading } = useRefreshQuery(null)
	const dispatch = useAppDispatch()

	const token = useAppSelector(getToken)
	const realm = useAppSelector(getRealm)

	useEffect(() => {
		if (isSuccess && data) {
			dispatch(setUser(data.data))
			if (!realm && data.data.realms.length > 0 && data.data.realms[0].realm) {
				dispatch(setRealm(data.data.realms[0].realm))
			}
		}
	}, [isSuccess, data, dispatch, realm])

	const isFinished = !isLoading && !isFetching
	const hasUserInStore = isSuccess && !!token

	const ready = isFinished && (isError || hasUserInStore)

	return { ready }
}
