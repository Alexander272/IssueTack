import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type {
	IEquipment,
	IEquipmentDTO,
	IEquipmentFilter,
	IEquipmentList,
	IEquipmentType,
	IEquipmentTypeDTO,
} from './types/equipment'
import { apiSlice } from '@/app/apiSlice'
import { API } from '@/app/api'

const equipmentApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getAllEquipment: builder.query<IEquipmentList, IEquipmentFilter | void>({
			query: filter => ({
				url: API.equipment.base,
				method: 'GET',
				params: filter ?? undefined,
			}),
			providesTags: [{ type: 'Equipment', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipment: builder.query<IEquipment, string>({
			query: id => ({
				url: API.equipment.byId(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'Equipment', id }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentChildren: builder.query<{ data: IEquipment[] }, string>({
			query: id => ({
				url: API.equipment.children(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'Equipment', id: `children-${id}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		createEquipment: builder.mutation<{ id: string; message: string }, IEquipmentDTO>({
			query: body => ({
				url: API.equipment.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'Equipment', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		updateEquipment: builder.mutation<{ id: string; message: string }, IEquipmentDTO>({
			query: body => ({
				url: API.equipment.byId(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: (_result, _err, body) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id: body.id! },
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

		deleteEquipment: builder.mutation<void, string>({
			query: id => ({
				url: API.equipment.byId(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'Equipment', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getAllEquipmentTypes: builder.query<{ data: IEquipmentType[] }, void>({
			query: () => ({
				url: API.equipment.types,
				method: 'GET',
			}),
			providesTags: [{ type: 'EquipmentTypes', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentType: builder.query<IEquipmentType, string>({
			query: id => ({
				url: API.equipment.typeById(id),
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

		createEquipmentType: builder.mutation<IEquipmentType, IEquipmentTypeDTO>({
			query: body => ({
				url: API.equipment.types,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'EquipmentTypes', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		updateEquipmentType: builder.mutation<IEquipmentType, IEquipmentTypeDTO>({
			query: body => ({
				url: API.equipment.typeById(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'EquipmentTypes', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		deleteEquipmentType: builder.mutation<void, string>({
			query: id => ({
				url: API.equipment.typeById(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'EquipmentTypes', id: 'ALL' }],
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
	useGetAllEquipmentQuery,
	useGetEquipmentQuery,
	useGetEquipmentChildrenQuery,
	useCreateEquipmentMutation,
	useUpdateEquipmentMutation,
	useDeleteEquipmentMutation,
	useGetAllEquipmentTypesQuery,
	useGetEquipmentTypeQuery,
	useCreateEquipmentTypeMutation,
	useUpdateEquipmentTypeMutation,
	useDeleteEquipmentTypeMutation,
} = equipmentApiSlice