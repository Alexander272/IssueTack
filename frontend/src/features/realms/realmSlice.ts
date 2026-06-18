import { type PayloadAction, createSlice } from '@reduxjs/toolkit'

import type { RootState } from '@/app/store'
import type { IRealm } from './types/realm'
import { STORAGE_KEYS } from '@/constants/storage'

interface IRealmState {
	realm: IRealm | null
}

const initialState: IRealmState = {
	realm: JSON.parse(localStorage.getItem(STORAGE_KEYS.ActiveRealm) || 'null'),
}

const realmSlice = createSlice({
	name: 'realm',
	initialState,
	reducers: {
		setRealm: (state, action: PayloadAction<IRealm>) => {
			state.realm = action.payload
			if (!state.realm) return
			localStorage.setItem(STORAGE_KEYS.ActiveRealm, JSON.stringify(state.realm))
		},

		resetRealm: () => initialState,
	},
})

export const realmPath = realmSlice.name
export const realmReducer = realmSlice.reducer

export const getRealm = (state: RootState) => state[realmPath].realm

export const { setRealm, resetRealm } = realmSlice.actions
