import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IUser } from '@/features/user/types/user'
import type { ISignIn } from './types/auth'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import { resetUser } from '../user/userSlice'

const authApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		signIn: builder.mutation<{ data: IUser }, ISignIn>({
			query: data => ({
				url: API.auth.signIn,
				method: 'POST',
				body: data,
			}),
			// onQueryStarted: async (_arg, api) => {
			// 	try {
			// 		await api.queryFulfilled
			// 	} catch (error) {
			// 		const fetchError = (error as IBaseFetchError).error
			// 		toast.error(fetchError.data.message, { autoClose: false })
			// 	}
			// },
		}),

		signOut: builder.mutation<null, null>({
			query: () => ({
				url: API.auth.signOut,
				method: 'POST',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
					api.dispatch(resetUser())
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		refresh: builder.query<{ data: IUser }, null>({
			query: () => ({
				url: API.auth.refresh,
				method: 'POST',
			}),
		}),
	}),
})

export const { useSignInMutation, useSignOutMutation, useRefreshQuery } = authApiSlice
