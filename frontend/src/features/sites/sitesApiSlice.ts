import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ISite, ISiteDTO } from './types/site'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const sitesApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllSites: builder.query<{ data: ISite[] }, void>({
			query: () => ({
				url: API.sites.base,
				method: 'GET',
			}),
			providesTags: [{ type: 'Sites', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		getSite: builder.query<ISite, string>({
			query: id => ({
				url: API.sites.byId(id),
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

		createSite: builder.mutation<ISite, ISiteDTO>({
			query: body => ({
				url: API.sites.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Sites', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		updateSite: builder.mutation<ISite, ISiteDTO>({
			query: body => ({
				url: API.sites.byId(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'Sites', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		deleteSite: builder.mutation<void, string>({
			query: id => ({
				url: API.sites.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Sites', id: 'ALL' }],
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

export const {
	useGetAllSitesQuery,
	useGetSiteQuery,
	useCreateSiteMutation,
	useUpdateSiteMutation,
	useDeleteSiteMutation,
} = sitesApiSlice
