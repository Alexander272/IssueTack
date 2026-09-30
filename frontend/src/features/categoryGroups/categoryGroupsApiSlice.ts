import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ICategoryGroup, ICategoryGroupDTO } from './types/categoryGroup'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const categoryGroupsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllCategoryGroups: builder.query<{ data: ICategoryGroup[] }, void>({
			query: () => ({
				url: API.categoryGroups.base,
				method: 'GET',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
			providesTags: [{ type: 'CategoryGroups', id: 'ALL' }],
		}),

		getCategoryGroup: builder.query<ICategoryGroup, string>({
			query: id => ({
				url: API.categoryGroups.byId(id),
				method: 'GET',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		createCategoryGroup: builder.mutation<ICategoryGroup, ICategoryGroupDTO>({
			query: body => ({
				url: API.categoryGroups.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'CategoryGroups', id: 'ALL' }],
		}),

		updateCategoryGroup: builder.mutation<ICategoryGroup, ICategoryGroupDTO>({
			query: body => ({
				url: API.categoryGroups.byId(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'CategoryGroups', id: 'ALL' }],
		}),

		deleteCategoryGroup: builder.mutation<void, string>({
			query: id => ({
				url: API.categoryGroups.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'CategoryGroups', id: 'ALL' }],
		}),
	}),
})

export const {
	useGetAllCategoryGroupsQuery,
	useGetCategoryGroupQuery,
	useCreateCategoryGroupMutation,
	useUpdateCategoryGroupMutation,
	useDeleteCategoryGroupMutation,
} = categoryGroupsApiSlice
