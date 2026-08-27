export interface IRealmMattermost {
	realmId: string
	botToken: string
	botUserId: string
	channelId: string
	webhookSecret: string
	isActive: boolean
	createdAt: string
	updatedAt: string
}

export interface IRealmMattermostDTO {
	botToken: string
	channelId: string
}
