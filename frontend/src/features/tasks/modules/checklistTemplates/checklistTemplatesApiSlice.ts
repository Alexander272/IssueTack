import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import type {
	IChecklistTemplate,
	IChecklistTemplateDTO,
	IChecklistTemplateItem,
	IChecklistTemplateItemDTO,
} from './types'

const checklistTemplatesApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getChecklistTemplates: builder.query<{ data: IChecklistTemplate[]; total?: number }, void>({
			query: () => ({
				url: API.checklists.base,
				method: 'GET',
			}),
			providesTags: ['ChecklistTemplates'],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		getChecklistTemplateItems: builder.query<
			{ data: IChecklistTemplateItem[]; total?: number },
			string
		>({
			query: templateId => ({
				url: API.checklists.items(templateId),
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

		createChecklistTemplate: builder.mutation<
			{ id: string; message: string },
			IChecklistTemplateDTO
		>({
			query: body => ({
				url: API.checklists.base,
				method: 'POST',
				body,
			}),
			invalidatesTags: ['ChecklistTemplates'],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),

		setChecklistTemplateItems: builder.mutation<
			{ message: string },
			{ id: string; items: IChecklistTemplateItemDTO[] }
		>({
			query: ({ id, items }) => ({
				url: API.checklists.items(id),
				method: 'PUT',
				body: items,
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
	}),
})

export const {
	useGetChecklistTemplatesQuery,
	useLazyGetChecklistTemplateItemsQuery,
	useCreateChecklistTemplateMutation,
	useSetChecklistTemplateItemsMutation,
} = checklistTemplatesApiSlice