import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IGroupedPermission } from './types/permission'
import type { IResource } from './types/resource'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export const permsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getPermissions: builder.query<{ data: IGroupedPermission[] }, null>({
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

		getResources: builder.query<{ data: IResource[] }, null>({
			query: () => ({
				url: API.permissions.resources,
				method: 'GET',
			}),
			providesTags: [{ type: 'Perms', id: 'Resources' }],
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

export const { useGetPermissionsQuery, useGetResourcesQuery } = permsApiSlice
