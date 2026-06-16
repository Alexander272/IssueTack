import { type PayloadAction, createSlice } from '@reduxjs/toolkit'

import type { RootState } from '@/app/store'
import type { IUser } from './types/user'

interface IUserState {
	// user?: IUser

	id: string | null
	name?: string
	role: string | null
	permissions: string[]
	token: string | null
}

const initialState: IUserState = {
	id: null,
	role: null,
	token: null,
	permissions: [],
}

const userSlice = createSlice({
	name: 'user',
	initialState,
	reducers: {
		setUser: (state, action: PayloadAction<IUser>) => {
			// state.user = action.payload

			state.id = action.payload.id
			state.name = action.payload.name
			state.role = action.payload.role
			state.permissions = action.payload.permissions
			state.token = action.payload.token
		},

		setRole: (state, action: PayloadAction<string>) => {
			state.role = action.payload
		},
		setPermissions: (state, action: PayloadAction<string[]>) => {
			state.permissions = action.payload
		},

		resetUser: () => initialState,
	},
})

export const getToken = (state: RootState) => state.user.token
export const getPermissions = (state: RootState) => state.user.permissions
export const getRole = (state: RootState) => state.user.role

export const userPath = userSlice.name
export const userReducer = userSlice.reducer

export const { setUser, setRole, setPermissions, resetUser } = userSlice.actions
