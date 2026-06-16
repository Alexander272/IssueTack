import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IUserData, IUserDataDTO, IUserLogin } from './types/user'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export const usersApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllUsers: builder.query<{ data: IUserData[] }, null>({
			query: () => ({
				url: API.users.base,
				method: 'GET',
			}),
			providesTags: [{ type: 'Users', id: 'All' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
		getUserByAccess: builder.query<{ data: IUserData[] }, null>({
			query: () => ({
				url: `${API.users.access}`,
				method: 'GET',
			}),
			providesTags: [
				{ type: 'Users', id: 'All' },
				{ type: 'Users', id: 'access' },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		getUserLogins: builder.query<{ data: IUserLogin[] }, string>({
			query: id => ({
				url: `${API.users.logins}/${id}`,
				method: 'GET',
			}),
		}),

		syncUsers: builder.mutation<null, null>({
			query: () => ({
				url: API.users.sync,
				method: 'POST',
			}),
			invalidatesTags: [{ type: 'Users', id: 'All' }],
		}),

		updateUser: builder.mutation<null, IUserDataDTO>({
			query: user => ({
				url: `${API.users.base}/${user.id}`,
				method: 'PUT',
				body: user,
			}),
			invalidatesTags: [{ type: 'Users', id: 'All' }],
		}),
	}),
})

export const {
	useGetAllUsersQuery,
	useGetUserByAccessQuery,
	useGetUserLoginsQuery,
	useSyncUsersMutation,
	useUpdateUserMutation,
} = usersApiSlice
