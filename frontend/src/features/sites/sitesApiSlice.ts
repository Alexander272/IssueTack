import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { ISite, ISiteDTO, ISiteLocation, ISiteLocationDTO } from './types/site'
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
					toast.error(fetchError.data?.message, { autoClose: false })
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
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		// Плоский список для селектора «Место»: площадки приходят отдельным
		// getAllSites, а локации берутся все сразу, без N-запросов на форму.
		getAllSiteLocations: builder.query<{ data: ISiteLocation[] }, void>({
			query: () => ({
				url: API.sites.allLocations,
				method: 'GET',
			}),
			providesTags: [{ type: 'SiteLocations', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getSiteLocations: builder.query<{ data: ISiteLocation[] }, string>({
			query: siteId => ({
				url: API.sites.locations(siteId),
				method: 'GET',
			}),
			providesTags: (_result, _err, siteId) => [
				{ type: 'SiteLocations', id: 'ALL' },
				{ type: 'SiteLocations', id: siteId },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		createSiteLocation: builder.mutation<{ message: string }, { siteId: string; body: ISiteLocationDTO }>({
			query: ({ siteId, body }) => ({
				url: API.sites.locations(siteId),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { siteId }) => [
				{ type: 'SiteLocations', id: 'ALL' },
				{ type: 'SiteLocations', id: siteId },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		updateSiteLocation: builder.mutation<
			{ message: string },
			{ siteId: string; locationId: string; body: ISiteLocationDTO }
		>({
			query: ({ siteId, locationId, body }) => ({
				url: API.sites.locationById(siteId, locationId),
				method: 'PUT',
				body,
			}),
			invalidatesTags: (_result, _err, { siteId }) => [
				{ type: 'SiteLocations', id: 'ALL' },
				{ type: 'SiteLocations', id: siteId },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		// Конфликт удаления (EQ025 — расположение ещё указано у оборудования,
		// в выдачах или актах) приходит с текстом бэкенда: onQueryStarted показывает его
		// тостом, а диалог подтверждения остаётся открытым.
		deleteSiteLocation: builder.mutation<void, { siteId: string; locationId: string }>({
			query: ({ siteId, locationId }) => ({
				url: API.sites.locationById(siteId, locationId),
				method: 'DELETE',
			}),
			invalidatesTags: (_result, _err, { siteId }) => [
				{ type: 'SiteLocations', id: 'ALL' },
				{ type: 'SiteLocations', id: siteId },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
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
					toast.error(fetchError.data?.message, { autoClose: false })
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
					toast.error(fetchError.data?.message, { autoClose: false })
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
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),
	}),
})

export const {
	useGetAllSitesQuery,
	useGetSiteQuery,
	useGetAllSiteLocationsQuery,
	useGetSiteLocationsQuery,
	useCreateSiteLocationMutation,
	useUpdateSiteLocationMutation,
	useDeleteSiteLocationMutation,
	useCreateSiteMutation,
	useUpdateSiteMutation,
	useDeleteSiteMutation,
} = sitesApiSlice
