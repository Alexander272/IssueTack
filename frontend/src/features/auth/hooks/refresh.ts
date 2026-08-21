import { useEffect } from 'react'

import { useAppDispatch, useAppSelector } from '@/hooks/redux'
import { getToken, setCapabilities, setUser } from '@/features/user/userSlice'
import { useGetCapabilitiesQuery } from '@/features/user/usersApiSlice'
import { getRealm, setRealm } from '@/features/realms/realmSlice'
import { useRefreshQuery } from '../authApiSlice'
import { toast } from 'react-toastify'

export function useRefresh() {
	const { data, isSuccess, isError, isFetching, isLoading } = useRefreshQuery(null)
	const dispatch = useAppDispatch()

	const token = useAppSelector(getToken)
	const realm = useAppSelector(getRealm)

	const {
		data: capsData,
		isSuccess: capsSuccess,
		isError: capsError,
	} = useGetCapabilitiesQuery(undefined, {
		skip: !token,
	})

	useEffect(() => {
		if (isSuccess && data) {
			dispatch(setUser(data.data))
			if (!realm && data.data.realms.length > 0 && data.data.realms[0].realm) {
				dispatch(setRealm(data.data.realms[0].realm))
			}
		}
	}, [isSuccess, data, dispatch, realm])

	useEffect(() => {
		if (capsSuccess && capsData) {
			dispatch(setCapabilities(capsData.data))
		}
	}, [capsSuccess, capsData, dispatch])

	useEffect(() => {
		if (capsError) {
			toast.error('Не удалось загрузить права доступа. Некоторые функции могут быть недоступны.')
		}
	}, [capsError])

	const isFinished = !isLoading && !isFetching
	const hasUserInStore = isSuccess && !!token

	const ready = isFinished && (isError || hasUserInStore)

	return { ready }
}
