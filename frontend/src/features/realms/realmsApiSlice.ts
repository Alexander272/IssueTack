import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IRealm, IRealmDTO } from './types/realm'
import type { IRealmMattermost, IRealmMattermostDTO } from './types/mattermost'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const realmsApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllRealms: builder.query<{ data: IRealm[] }, void>({
			query: () => ({
				url: API.realms.base,
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
			providesTags: [{ type: 'Realms', id: 'ALL' }],
		}),

		getRealm: builder.query<IRealm, string>({
			query: id => ({
				url: API.realms.byId(id),
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

		createRealm: builder.mutation<IRealm, Omit<IRealmDTO, 'id'>>({
			query: body => ({
				url: API.realms.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Realms', id: 'ALL' }],
		}),

		updateRealm: builder.mutation<IRealm, IRealmDTO>({
			query: body => ({
				url: API.realms.byId(body.id),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'Realms', id: 'ALL' }],
		}),

		deleteRealm: builder.mutation<void, string>({
			query: id => ({
				url: API.realms.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Realms', id: 'ALL' }],
		}),

		getMattermostSettings: builder.query<{ data: IRealmMattermost }, string>({
			query: realmId => ({
				url: API.realms.mattermost(realmId),
				method: 'GET',
			}),
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = error as IBaseFetchError
					if (fetchError.error?.status !== 404) {
						toast.error(fetchError.error.data.message, { autoClose: false })
					}
				}
			},
			providesTags: [{ type: 'Mattermost', id: 'ALL' }],
		}),

		saveMattermostSettings: builder.mutation<void, { realmId: string; dto: IRealmMattermostDTO }>({
			query: ({ realmId, dto }) => ({
				url: API.realms.mattermost(realmId),
				method: 'PUT',
				body: dto,
			}),
			invalidatesTags: [{ type: 'Mattermost', id: 'ALL' }],
		}),

		deleteMattermostSettings: builder.mutation<void, string>({
			query: realmId => ({
				url: API.realms.mattermost(realmId),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Mattermost', id: 'ALL' }],
		}),
	}),
})

export const {
	useGetAllRealmsQuery,
	useGetRealmQuery,
	useCreateRealmMutation,
	useUpdateRealmMutation,
	useDeleteRealmMutation,
	useGetMattermostSettingsQuery,
	useSaveMattermostSettingsMutation,
	useDeleteMattermostSettingsMutation,
} = realmsApiSlice
