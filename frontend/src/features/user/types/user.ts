import type { IRealm } from '@/features/realms/types/realm'
import type { IRole } from './role'

export interface IUser {
	id: string
	name: string
	role: string
	permissions: Record<string, string[]>
	token: string
	realms: IUserRealm[]
}

export interface IUserShort {
	id: string
	username: string
	firstName: string
	lastName: string
	email: string
	internalNumber?: string
}

export interface IUserData {
	id: string
	username: string
	firstName: string
	lastName: string
	email: string
	mattermostId: string
	isActive: boolean
	createdAt: string

	realms: IUserRealm[]
}

export interface IUserRealm {
	id: string | null
	userId: string
	realmId: string
	roleId: string
	realm?: IRealm
	role?: IRole
	isActive: boolean
	createdAt: string
}

export interface IUserDataDTO {
	id: string
	username: string
	firstName: string
	lastName: string
	email: string
	mattermostId: string
	isActive: boolean

	realms: IUserRealm[]
}

export interface IUserLogin {
	id: string
	userId: string
	loginAt: string
	ipAddress: string | null
	userAgent: string | null
	metadata?: IUserMetadata
	lastActivityAt: string
}
export interface IUserMetadata {
	browser: string
	browserVersion: string
	device: 'desktop' | 'mobile' | 'tablet'
	event: string
	isBot: boolean
	isDesktop: boolean
	isMobile: boolean
	isTablet: boolean
	os: string
	osVersion: string
	success: boolean
}
