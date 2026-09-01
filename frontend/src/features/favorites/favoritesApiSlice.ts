import { toast } from 'react-toastify'

import type { IBaseFetchError } from '@/app/types/error'
import { API } from '@/app/api'
import { apiSlice } from '@/app/apiSlice'
import type { FavoriteType, IFavoriteState } from './types/favorite'

interface FavoriteMutationArgs {
	id: string
	type: FavoriteType
}

const favoritesApiSlice = apiSlice.injectEndpoints({
	overrideExisting: false,
	endpoints: builder => ({
		getFavoriteState: builder.query<{ data: IFavoriteState }, string>({
			query: id => ({
				url: API.favorites.byTicket(id),
				method: 'GET',
			}),
			providesTags: (_result, _error, id) => [{ type: 'Favorites', id: `state:${id}` }],
		}),
		addFavorite: builder.mutation<{ data: { message: string } }, FavoriteMutationArgs>({
			query: ({ id, type }) => ({
				url: API.favorites.byTicket(id),
				method: 'POST',
				body: { type },
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Favorites', id: 'LIST' },
				{ type: 'Favorites', id: `state:${arg.id}` },
				{ type: 'Tasks', id: 'LIST' },
			],
			onQueryStarted: async (_arg, api) => {
				try {
					await api.queryFulfilled
				} catch (error) {
					const fetchError = (error as IBaseFetchError).error
					toast.error(fetchError.data.message, { autoClose: false })
				}
			},
		}),
		removeFavorite: builder.mutation<{ data: { message: string } }, FavoriteMutationArgs>({
			query: ({ id, type }) => ({
				url: API.favorites.byTicket(id),
				method: 'DELETE',
				body: { type },
			}),
			invalidatesTags: (_result, _error, arg) => [
				{ type: 'Favorites', id: 'LIST' },
				{ type: 'Favorites', id: `state:${arg.id}` },
				{ type: 'Tasks', id: 'LIST' },
			],
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

export const { useGetFavoriteStateQuery, useAddFavoriteMutation, useRemoveFavoriteMutation } = favoritesApiSlice
