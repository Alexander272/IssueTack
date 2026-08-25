export interface ICommentUser {
	id: string
	username: string
	firstName?: string
	lastName?: string
}

export interface IComment {
	id: string
	text: string
	userId: string
	ticketId: string
	isInternal: boolean
	type: string
	createdAt: string
	user?: ICommentUser
}
