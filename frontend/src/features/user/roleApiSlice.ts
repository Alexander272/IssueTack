import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type { IRole, IRoleWithPerms, IRoleWithPermsDTO, IRoleWithStats } from './types/role'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'

export const rolesApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getRoles: builder.query<{ data: IRole[] }, null>({
			query: () => ({
				url: API.roles.base,
				method: 'GET',
			}),
			providesTags: [{ type: 'Roles', id: 'all' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
		getRoleWithPermissions: builder.query<{ data: IRoleWithPerms }, string>({
			query: id => ({
				url: API.roles.permissions(id),
				method: 'GET',
			}),
			providesTags: (_result, _error, id) => [{ type: 'Roles', id: id }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
		getRolesWithStats: builder.query<{ data: IRoleWithStats[] }, null>({
			query: () => ({
				url: API.roles.stats,
				method: 'GET',
			}),
			providesTags: [{ type: 'Roles', id: 'stats' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		createRole: builder.mutation<null, IRoleWithPermsDTO>({
			query: role => ({
				url: API.roles.base,
				method: 'POST',
				body: role,
			}),
			invalidatesTags: [{ type: 'Roles' }],
		}),
		updateRole: builder.mutation<null, IRoleWithPermsDTO>({
			query: role => ({
				url: `${API.roles.base}/${role.id}`,
				method: 'PUT',
				body: role,
			}),
			invalidatesTags: [{ type: 'Roles' }],
		}),
		deleteRole: builder.mutation<null, string>({
			query: id => ({
				url: `${API.roles.base}/${id}`,
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Roles' }],
		}),
	}),
})

export const {
	useGetRolesQuery,
	useGetRoleWithPermissionsQuery,
	useGetRolesWithStatsQuery,
	useCreateRoleMutation,
	useUpdateRoleMutation,
	useDeleteRoleMutation,
} = rolesApiSlice
