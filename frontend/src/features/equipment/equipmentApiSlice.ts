import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type {
	IAssignEquipmentDTO,
	IChangeStatusDTO,
	ICompositionOpDTO,
	IEquipment,
	IEquipmentAct,
	IEquipmentActDTO,
	IEquipmentActFilter,
	IEquipmentActList,
	IEquipmentAssignment,
	IEquipmentDTO,
	IEquipmentEvent,
	IEquipmentFilter,
	IEquipmentList,
	IEquipmentType,
	IEquipmentTypeDTO,
	IKitComposition,
	IUnassignEquipmentDTO,
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

		assignEquipment: builder.mutation<{ id: string; message: string }, { id: string; body: IAssignEquipmentDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.assign(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
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

		unassignEquipment: builder.mutation<{ id: string; message: string }, { id: string; body: IUnassignEquipmentDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.unassign(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
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

		changeEquipmentStatus: builder.mutation<IEquipment, { id: string; body: IChangeStatusDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.status(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
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

		getEquipmentAssignments: builder.query<{ data: IEquipmentAssignment[] }, string>({
			query: id => ({
				url: API.equipment.assignments(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'Equipment', id: `assignments-${id}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentEvents: builder.query<{ data: IEquipmentEvent[] }, { id: string; limit?: number; offset?: number }>({
			query: ({ id, ...params }) => ({
				url: API.equipment.events(id),
				method: 'GET',
				params,
			}),
			providesTags: (_result, _err, { id }) => [{ type: 'Equipment', id: `events-${id}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentComposition: builder.query<{ data: IEquipment[] }, string>({
			query: id => ({
				url: API.equipment.composition(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'Equipment', id: `composition-${id}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentCompositionHistory: builder.query<{ data: IKitComposition[] }, string>({
			query: id => ({
				url: API.equipment.compositionHistory(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'Equipment', id: `composition-history-${id}` }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		addEquipmentComponents: builder.mutation<{ id: string; message: string }, { id: string; body: ICompositionOpDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.composition(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
				{ type: 'Equipment', id: `composition-${id}` },
				{ type: 'Equipment', id: `composition-history-${id}` },
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

		removeEquipmentComponents: builder.mutation<{ id: string; message: string }, { id: string; body: ICompositionOpDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.composition(id),
				method: 'DELETE',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
				{ type: 'Equipment', id: `composition-${id}` },
				{ type: 'Equipment', id: `composition-history-${id}` },
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

		getAllEquipmentActs: builder.query<IEquipmentActList, IEquipmentActFilter | void>({
			query: filter => ({
				url: API.equipment.acts,
				method: 'GET',
				params: filter ?? undefined,
			}),
			providesTags: [{ type: 'EquipmentActs', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentAct: builder.query<IEquipmentAct, string>({
			query: id => ({
				url: API.equipment.actById(id),
				method: 'GET',
			}),
			providesTags: (_result, _err, id) => [{ type: 'EquipmentActs', id }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		createEquipmentAct: builder.mutation<{ id: string; message: string }, IEquipmentActDTO>({
			query: body => ({
				url: API.equipment.acts,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'EquipmentActs', id: 'ALL' }],
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
	useAssignEquipmentMutation,
	useUnassignEquipmentMutation,
	useChangeEquipmentStatusMutation,
	useGetEquipmentAssignmentsQuery,
	useGetEquipmentEventsQuery,
	useGetEquipmentCompositionQuery,
	useGetEquipmentCompositionHistoryQuery,
	useAddEquipmentComponentsMutation,
	useRemoveEquipmentComponentsMutation,
	useGetAllEquipmentActsQuery,
	useGetEquipmentActQuery,
	useCreateEquipmentActMutation,
	useGetAllEquipmentTypesQuery,
	useGetEquipmentTypeQuery,
	useCreateEquipmentTypeMutation,
	useUpdateEquipmentTypeMutation,
	useDeleteEquipmentTypeMutation,
} = equipmentApiSlice