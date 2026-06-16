export interface IUser {
	id: string
	name: string
	role: string
	permissions: string[]
	token: string
}

export interface IUserShort {
	id: string
	ssoId: string
	firstName: string
	lastName: string
	email: string
}

export interface IUserData {
	id: string
	ssoId: string
	username: string
	firstName: string
	lastName: string
	email: string
	roleId: string
	role: string
	isActive: boolean
	createdAt: string
	lastVisit: string
}

export interface IUserDataDTO {
	id: string
	ssoId: string
	roleId: string
	username: string
	firstName: string
	lastName: string
	email: string
	isActive: boolean
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
