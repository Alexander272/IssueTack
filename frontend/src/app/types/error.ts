export interface IFetchError {
	data: {
		message: string
		code: string
		fields?: IFieldError[]
	}
	status: number
}

export interface IFieldError {
	field: string
	message: string
	tag?: string
}

export interface IBaseFetchError {
	error: {
		data: {
			message: string
			code: string
		}
		status: number
	}
	status: number
}
