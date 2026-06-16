import { Link } from 'react-router'
import { Button, Divider, Stack, Typography, useTheme } from '@mui/material'

import { Container } from './notFound.style'
import logo from '@/assets/logo192.webp'

export default function NotFound() {
	const { palette } = useTheme()

	return (
		<Container>
			<Stack
				direction={'row'}
				divider={<Divider orientation='vertical' flexItem />}
				spacing={4}
				sx={{ alignItems: 'center' }}
			>
				<img width={148} height={148} src={logo} alt='logo' />
				<Typography
					variant='h4'
					sx={{
						fontSize: '8rem',
						fontWeight: 'bold',
						color: palette.common.white,
						WebkitTextStrokeWidth: 3,
						WebkitTextStrokeColor: palette.primary.main,
					}}
				>
					404
				</Typography>
			</Stack>

			<Typography sx={{ mt: 3, mb: 3, fontSize: '2rem', color: palette.primary.main }}>
				Страница не найдена
			</Typography>

			<Link to='/'>
				<Button variant='outlined' size='large' sx={{ borderRadius: '12px', padding: '8px 32px' }}>
					На главную
				</Button>
			</Link>
		</Container>
	)
}
