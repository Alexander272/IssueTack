import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ICategory, ICategoryDTO } from './types/category'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const categoriesApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllCategories: builder.query<{ data: ICategory[] }, void>({
			query: () => ({
				url: API.categories.base,
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
			providesTags: [{ type: 'Categories', id: 'ALL' }],
		}),

		getCategory: builder.query<ICategory, string>({
			query: id => ({
				url: API.categories.byId(id),
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

		createCategory: builder.mutation<ICategory, Omit<ICategoryDTO, 'id'>>({
			query: body => ({
				url: API.categories.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Categories', id: 'ALL' }],
		}),

		updateCategory: builder.mutation<ICategory, ICategoryDTO>({
			query: body => ({
				url: API.categories.byId(body.id),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'Categories', id: 'ALL' }],
		}),

		deleteCategory: builder.mutation<void, string>({
			query: id => ({
				url: API.categories.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Categories', id: 'ALL' }],
		}),
	}),
})

export const {
	useGetAllCategoriesQuery,
	useGetCategoryQuery,
	useCreateCategoryMutation,
	useUpdateCategoryMutation,
	useDeleteCategoryMutation,
} = categoriesApiSlice
