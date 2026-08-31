import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IUserData, IUserDataDTO, IUserCapabilities, IUserLogin } from './types/user'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export type MembershipFilter = 'all' | 'customers' | 'executors'

export const usersApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllUsers: builder.query<{ data: IUserData[] }, void>({
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
			getUserByAccess: builder.query<{ data: IUserData[] }, void>({
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

		getAvailableUsers: builder.query<{ data: IUserData[] }, void>({
			query: () => ({
				url: API.users.available,
				method: 'GET',
			}),
			providesTags: [{ type: 'Users', id: 'available' }],
		}),

		getRealmUsers: builder.query<{ data: IUserData[] }, MembershipFilter | void>({
			query: membership => {
				const params = membership && membership !== 'all' ? { membership } : undefined
				return { url: API.users.available, method: 'GET', params }
			},
			providesTags: [{ type: 'Users', id: 'available' }],
		}),

		getCapabilities: builder.query<{ data: Record<string, IUserCapabilities> }, void>({
			query: () => ({
				url: API.users.capabilities,
				method: 'GET',
			}),
			providesTags: [{ type: 'Users', id: 'capabilities' }],
		}),

		getUserLogins: builder.query<{ data: IUserLogin[] }, string>({
			query: id => ({
				url: `${API.users.logins}/${id}`,
				method: 'GET',
			}),
		}),

		syncUsers: builder.mutation<null, void>({
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
	useGetAvailableUsersQuery,
	useGetRealmUsersQuery,
	useGetCapabilitiesQuery,
	useSyncUsersMutation,
	useUpdateUserMutation,
} = usersApiSlice
