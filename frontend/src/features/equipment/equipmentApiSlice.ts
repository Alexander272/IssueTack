import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import type {
	IAddDefectDTO,
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
	IEquipmentKitTemplate,
	IEquipmentKitTemplateDTO,
	IEquipmentList,
	IEquipmentShop,
	IEquipmentStats,
	IEquipmentTreeArgs,
	IEquipmentTreeList,
	IEquipmentType,
	IEquipmentTypeDTO,
	IKitComposition,
	IUnassignEquipmentDTO,
	IWriteOffEquipmentDTO,
} from './types/equipment'
import { apiSlice, baseQueryWithReAuth } from '@/app/apiSlice'
import { API } from '@/app/api'
import { saveAs } from '@/utils/saveAs'

// signedFileUrls — blob-URL подписанных актов по id акта. Держим их здесь (а не в
// состоянии компонента), чтобы карточка и кнопка скачивания делили один URL, а
// отзыв шёл по удалению записи из кэша RTK Query.
const signedFileUrls = new Map<string, string>()

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

		getEquipment: builder.query<{ data: IEquipment }, string>({
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

		getEquipmentStats: builder.query<{ data: IEquipmentStats }, void>({
			query: () => ({
				url: API.equipment.stats,
				method: 'GET',
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

		getEquipmentTree: builder.query<IEquipmentTreeList, IEquipmentTreeArgs | void>({
			query: args => ({
				url: API.equipment.tree,
				method: 'GET',
				params: args ?? undefined,
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

		getEquipmentChildren: builder.query<{ data: IEquipment[] }, string>({
			query: id => ({
				url: API.equipment.children(id),
				method: 'GET',
			}),
			// Список детей слушает и общий тег ALL: состав меняется не только самой
			// единицей, а апдейтом ребёнка (прикрепили/открепили составную часть),
			// поэтому без ALL раздел «Физический состав» оставался бы протухшим.
			providesTags: (_result, _err, id) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id: `children-${id}` },
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
			// Выдача может сформировать акт (createAct), поэтому список актов тоже
			// устаревает: без этого тега новый акт появился бы на вкладке только
			// после перезагрузки страницы.
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
				{ type: 'EquipmentActs', id: 'ALL' },
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

		unassignEquipment: builder.mutation<
			{ id: string; message: string },
			{ id: string; body: IUnassignEquipmentDTO }
		>({
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

		addEquipmentDefect: builder.mutation<IEquipment, { id: string; body: IAddDefectDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.defect(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
				{ type: 'Equipment', id: `events-${id}` },
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

		writeOffEquipment: builder.mutation<IEquipment, { id: string; body: IWriteOffEquipmentDTO }>({
			query: ({ id, body }) => ({
				url: API.equipment.writeOff(id),
				method: 'POST',
				body,
			}),
			invalidatesTags: (_result, _err, { id }) => [
				{ type: 'Equipment', id: 'ALL' },
				{ type: 'Equipment', id },
				{ type: 'Equipment', id: `events-${id}` },
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

		getEquipmentEvents: builder.query<{ data: IEquipmentEvent[] }, { id: string; limit?: number; offset?: number }>(
			{
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
			},
		),

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

		addEquipmentComponents: builder.mutation<
			{ id: string; message: string },
			{ id: string; body: ICompositionOpDTO }
		>({
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

		removeEquipmentComponents: builder.mutation<
			{ id: string; message: string },
			{ id: string; body: ICompositionOpDTO }
		>({
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

		getEquipmentAct: builder.query<{ data: IEquipmentAct }, string>({
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

		// Документ акта и подписанный файл — бинарные ответы, поэтому их читает
		// queryFn с blob-обработчиком (как вложения заявок): так срабатывает
		// обновление токена, скачивание инициируется здесь же
		getEquipmentActDocument: builder.query<null, { id: string; filename: string }>({
			queryFn: async ({ id, filename }, api, extraOptions) => {
				const res = await baseQueryWithReAuth(
					{
						url: API.equipment.actDocument(id),
						responseHandler: (response: Response) => response.blob(),
						cache: 'no-cache',
					},
					api,
					extraOptions,
				)
				if (res.error) return { error: res.error }

				if (res.data instanceof Blob) {
					saveAs(res.data, filename)
				}
				return { data: null }
			},
		}),

		// Подписанный акт нужен и для предпросмотра (img src), поэтому в отличие от
		// документа он возвращает blob-URL, а не запускает скачивание: Blob при этом
		// в Redux не попадает — в кэш уходит только строка URL.
		getEquipmentActSignedFile: builder.query<{ url: string; size: number }, string>({
			queryFn: async (id, api, extraOptions) => {
				const res = await baseQueryWithReAuth(
					{
						url: API.equipment.actSigned(id),
						responseHandler: (response: Response) => response.blob(),
					},
					api,
					extraOptions,
				)
				if (res.error) return { error: res.error }

				const blob = res.data as Blob
				const url = URL.createObjectURL(blob)
				const prev = signedFileUrls.get(id)
				signedFileUrls.set(id, url)
				// Отзываем старый URL на следующем тике, чтобы живой рендер (превью)
				// успел переключиться на новый blob-URL; мгновенный revoke оборвал бы его.
				if (prev) setTimeout(() => URL.revokeObjectURL(prev), 0)
				return { data: { url, size: blob.size } }
			},
			onCacheEntryAdded: async (id, { cacheEntryRemoved }) => {
				await cacheEntryRemoved
				const url = signedFileUrls.get(id)
				if (url) URL.revokeObjectURL(url)
				signedFileUrls.delete(id)
			},
		}),

		uploadEquipmentActSigned: builder.mutation<{ id: string; message: string }, { id: string; file: File }>({
			query: ({ id, file }) => {
				const formData = new FormData()
				formData.append('file', file)
				return {
					url: API.equipment.actSigned(id),
					method: 'POST',
					body: formData,
				}
			},
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

		getAllEquipmentShops: builder.query<{ data: IEquipmentShop[] }, void>({
			query: () => ({
				url: API.equipment.shops,
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

		getAllEquipmentTemplates: builder.query<{ data: IEquipmentKitTemplate[] }, void>({
			query: () => ({
				url: API.equipment.templates,
				method: 'GET',
			}),
			providesTags: [{ type: 'EquipmentTemplates', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		getEquipmentTemplate: builder.query<{ data: IEquipmentKitTemplate }, string>({
			query: id => ({
				url: API.equipment.templateById(id),
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

		createEquipmentTemplate: builder.mutation<{ id: string; message: string }, IEquipmentKitTemplateDTO>({
			query: body => ({
				url: API.equipment.templates,
				method: 'POST',
				body,
			}),
			invalidatesTags: [{ type: 'EquipmentTemplates', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		updateEquipmentTemplate: builder.mutation<{ id: string; message: string }, IEquipmentKitTemplateDTO>({
			query: body => ({
				url: API.equipment.templateById(body.id!),
				method: 'PUT',
				body,
			}),
			invalidatesTags: [{ type: 'EquipmentTemplates', id: 'ALL' }],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data?.message, { autoClose: false })
				}
			},
		}),

		deleteEquipmentTemplate: builder.mutation<void, string>({
			query: id => ({
				url: API.equipment.templateById(id),
				method: 'DELETE',
			}),
			invalidatesTags: [{ type: 'EquipmentTemplates', id: 'ALL' }],
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
	useGetEquipmentStatsQuery,
	useGetEquipmentChildrenQuery,
	useGetEquipmentTreeQuery,
	useCreateEquipmentMutation,
	useUpdateEquipmentMutation,
	useDeleteEquipmentMutation,
	useAssignEquipmentMutation,
	useUnassignEquipmentMutation,
	useChangeEquipmentStatusMutation,
	useAddEquipmentDefectMutation,
	useWriteOffEquipmentMutation,
	useGetEquipmentAssignmentsQuery,
	useGetEquipmentEventsQuery,
	useGetEquipmentCompositionQuery,
	useGetEquipmentCompositionHistoryQuery,
	useAddEquipmentComponentsMutation,
	useRemoveEquipmentComponentsMutation,
	useGetAllEquipmentActsQuery,
	useGetEquipmentActQuery,
	useLazyGetEquipmentActQuery,
	useGetEquipmentActSignedFileQuery,
	useCreateEquipmentActMutation,
	useLazyGetEquipmentActDocumentQuery,
	useLazyGetEquipmentActSignedFileQuery,
	useUploadEquipmentActSignedMutation,
	useGetAllEquipmentTypesQuery,
	useGetAllEquipmentShopsQuery,
	useGetEquipmentTypeQuery,
	useCreateEquipmentTypeMutation,
	useUpdateEquipmentTypeMutation,
	useDeleteEquipmentTypeMutation,
	useGetAllEquipmentTemplatesQuery,
	useGetEquipmentTemplateQuery,
	useCreateEquipmentTemplateMutation,
	useUpdateEquipmentTemplateMutation,
	useDeleteEquipmentTemplateMutation,
} = equipmentApiSlice
