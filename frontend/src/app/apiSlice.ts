import { Mutex } from 'async-mutex'
import {
	type BaseQueryFn,
	type FetchArgs,
	type FetchBaseQueryError,
	createApi,
	fetchBaseQuery,
} from '@reduxjs/toolkit/query/react'

import type { IUser } from '@/features/user/types/user'
import type { RootState } from './store'
import { BaseUrl } from '@/constants/baseUrl'
import { resetUser, setUser } from '@/features/user/userSlice'
import { API } from './api'

const baseQuery = fetchBaseQuery({
	baseUrl: BaseUrl,
	mode: 'cors',
	credentials: 'include',
	prepareHeaders: (headers, api) => {
		const token = (api.getState() as RootState).user.token
		if (token) headers.set('authorization', `Bearer ${token}`)

		const realm = (api.getState() as RootState).realm.realm
		if (realm) headers.set('realm', realm.id)

		return headers
	},
})

const mutex = new Mutex()

type BaseQuery = BaseQueryFn<string | FetchArgs, unknown, FetchBaseQueryError>
const baseQueryWithReAuth: BaseQuery = async (args, api, extraOptions) => {
	// mutex позволяет предотвратить множественное обращение на обновление токена
	await mutex.waitForUnlock()
	let result = await baseQuery(args, api, extraOptions)

	if (result.error && result.error.status === 401 && api.endpoint !== 'signIn' && api.endpoint != 'refresh') {
		if (!mutex.isLocked()) {
			const release = await mutex.acquire()

			try {
				const refreshResult = await baseQuery({ url: API.auth.refresh, method: 'POST' }, api, extraOptions)
				if (refreshResult.data) {
					api.dispatch(setUser((refreshResult.data as { data: IUser }).data))
					result = await baseQuery(args, api, extraOptions)
				} else {
					api.dispatch(resetUser())
				}
			} finally {
				release()
			}
		} else {
			await mutex.waitForUnlock()
			result = await baseQuery(args, api, extraOptions)
		}
	}
	return result
}

export const apiSlice = createApi({
	reducerPath: 'api',
	baseQuery: baseQueryWithReAuth,
	tagTypes: ['Tasks', 'Users', 'Roles', 'Perms', 'AuditLogs', 'ActivityLogs', 'SearchLogs', 'Realms', 'Categories', 'Groups'],
	endpoints: () => ({}),
})
