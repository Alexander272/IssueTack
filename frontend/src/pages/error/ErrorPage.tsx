import { useRouteError, isRouteErrorResponse } from 'react-router'

import { ErrorState } from '@/components/ErrorState/ErrorState'
import { NotFound } from '../notFound/NotFoundLazy'

export default function ErrorPage() {
	const error = useRouteError()

	if (isRouteErrorResponse(error) && error.status === 404) {
		return <NotFound />
	}

	const status = isRouteErrorResponse(error) ? error.status : 500
	const message = isRouteErrorResponse(error)
		? error.statusText
		: error instanceof Error
			? error.message
			: 'Неизвестная ошибка'

	return (
		<ErrorState
			status={status}
			heading={status === 500 ? 'Внутренняя ошибка' : `Ошибка ${status}`}
			description={message}
		/>
	)
}
