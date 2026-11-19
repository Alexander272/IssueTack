import { Link } from 'react-router'
import { Button, Divider, Stack, Typography, useTheme } from '@mui/material'
import styled from '@emotion/styled'

import logo from '@/assets/logo192.webp'

const Container = styled.div`
	max-width: 1580px;
	height: 100vh;
	flex-grow: 1;
	display: flex;
	flex-direction: column;
	justify-content: center;
	align-items: center;
	flex-wrap: wrap;
	gap: 10px;
	margin: auto;
	width: 100%;
	min-height: 300px;
`

export interface ErrorStateProps {
	/** Статус или код ошибки, крупно отображаемый (403 / 404 / 500). */
	status: number
	/** Человекочитаемый заголовок ошибки. */
	heading: string
	/** Дополнительное описание (техническое сообщение ошибки). Необязательно. */
	description?: string
	/** Цвет акцента: контур статуса и заголовок. По умолчанию — цвет ошибки. */
	accentColor?: string
}

// Общий вид экранов ошибок: логотип, контурный статус, заголовок,
// опциональное описание и кнопка «На главную».
export const ErrorState = ({ status, heading, description, accentColor }: ErrorStateProps) => {
	const { palette } = useTheme()
	const accent = accentColor ?? palette.error.main

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
						WebkitTextStrokeColor: accent,
					}}
				>
					{status}
				</Typography>
			</Stack>

			<Typography sx={{ mt: 3, mb: 3, fontSize: '2rem', color: accent }}>{heading}</Typography>

			{description && (
				<Typography sx={{ fontSize: '0.95rem', color: palette.text.secondary, maxWidth: 600, textAlign: 'center', mb: 3 }}>
					{description}
				</Typography>
			)}

			<Link to='/'>
				<Button variant='outlined' size='large' sx={{ borderRadius: '12px', padding: '8px 32px' }}>
					На главную
				</Button>
			</Link>
		</Container>
	)
}
