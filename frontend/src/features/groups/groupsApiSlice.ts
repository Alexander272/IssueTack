import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IGroup, IGroupDTO } from './types/group'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const groupsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllGroups: builder.query<{ data: IGroup[] }, void>({
			query: () => ({
				url: API.groups.base,
				method: 'GET',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
			providesTags: [{ type: 'Groups', id: 'ALL' }],
		}),

		getGroup: builder.query<IGroup, string>({
			query: id => ({
				url: API.groups.byId(id),
				method: 'GET',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		createGroup: builder.mutation<IGroup, IGroupDTO>({
			query: body => ({
				url: API.groups.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Groups', id: 'ALL' }],
		}),

		updateGroup: builder.mutation<IGroup, IGroupDTO>({
			query: body => ({
				url: API.groups.byId(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'Groups', id: 'ALL' }],
		}),

		deleteGroup: builder.mutation<void, string>({
			query: id => ({
				url: API.groups.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Groups', id: 'ALL' }],
		}),
	}),
})

export const {
	useGetAllGroupsQuery,
	useGetGroupQuery,
	useCreateGroupMutation,
	useUpdateGroupMutation,
	useDeleteGroupMutation,
} = groupsApiSlice
