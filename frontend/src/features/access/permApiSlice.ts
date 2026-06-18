import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IGroupedPermission } from './types/permission'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export const permsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getPermissions: builder.query<{ data: IGroupedPermission[] }, void>({
			query: () => ({
				url: API.permissions.base,
				method: 'GET',
			}),
			providesTags: [{ type: 'Perms', id: 'All' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
	}),
})

export const { useGetPermissionsQuery } = permsApiSlice
