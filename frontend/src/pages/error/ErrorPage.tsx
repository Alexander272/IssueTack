import { Link } from 'react-router'
import { useRouteError, isRouteErrorResponse } from 'react-router'
import { Button, Divider, Stack, Typography, useTheme } from '@mui/material'

import { Container } from './errorPage.style'
import logo from '@/assets/logo192.webp'
import { NotFound } from '../notFound/NotFoundLazy'

export default function ErrorPage() {
	const error = useRouteError()
	const { palette } = useTheme()

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
		<Container>
			<Stack
				direction={'row'}
				divider={<Divider orientation='vertical' flexItem />}
				spacing={4}
				alignItems={'center'}
			>
				<img width={148} height={148} src={logo} alt='logo' />
				<Typography
					variant='h4'
					sx={{
						fontSize: '8rem',
						fontWeight: 'bold',
						color: palette.common.white,
						WebkitTextStrokeWidth: 3,
						WebkitTextStrokeColor: palette.error.main,
					}}
				>
					{status}
				</Typography>
			</Stack>
			<Typography mt={3} mb={1} sx={{ fontSize: '1.5rem', color: palette.error.main }}>
				{status === 500 ? 'Внутренняя ошибка клиента' : `Ошибка ${status}`}
			</Typography>
			<Typography
				mb={3}
				sx={{ fontSize: '0.95rem', color: palette.text.secondary, maxWidth: 600, textAlign: 'center' }}
			>
				{message}
			</Typography>
			<Link to='/'>
				<Button variant='outlined' size='large' sx={{ borderRadius: '12px', padding: '8px 32px' }}>
					На главную
				</Button>
			</Link>
		</Container>
	)
}
