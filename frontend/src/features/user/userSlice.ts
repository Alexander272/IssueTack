import { type PayloadAction, createSelector, createSlice } from '@reduxjs/toolkit'

import type { RootState } from '@/app/store'
import type { IUser, IUserRealm } from './types/user'
import { getRealm } from '../realms/realmSlice'

interface IUserState {
	id: string | null
	name?: string
	role: string | null
	permissions: Record<string, string[]>
	token: string | null
	realms: IUserRealm[]
}

const initialState: IUserState = {
	id: null,
	role: null,
	token: null,
	permissions: {},
	realms: [],
}

const userSlice = createSlice({
	name: 'user',
	initialState,
	reducers: {
		setUser: (state, action: PayloadAction<IUser>) => {
			state.id = action.payload.id
			state.name = action.payload.name
			// state.role = action.payload.realms[0].role?.name ||''
			state.permissions = action.payload.permissions
			state.token = action.payload.token
			state.realms = action.payload.realms
		},

		setRole: (state, action: PayloadAction<string>) => {
			state.role = action.payload
		},
		setPermissions: (state, action: PayloadAction<Record<string, string[]>>) => {
			state.permissions = action.payload
		},

		resetUser: () => initialState,
	},
})

export const getToken = (state: RootState) => state.user.token
export const getPermissions = (state: RootState) => state.user.permissions
export const getRole = (state: RootState) => state.user.role
export const getUserRealms = (state: RootState) => state.user.realms

export const getCurrentTenantPermissions = createSelector(
	[getPermissions, getRealm],
	(permissions, realm) => permissions[realm?.id || ''] ?? [],
)

export const getPermissionsSet = createSelector([getCurrentTenantPermissions], permissions => new Set(permissions))

export const userPath = userSlice.name
export const userReducer = userSlice.reducer

export const { setUser, setRole, setPermissions, resetUser } = userSlice.actions
