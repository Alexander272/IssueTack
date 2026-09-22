import { useEffect } from 'react'
import { Box, useTheme } from '@mui/material'
import { useLocation, useNavigate } from 'react-router'
import Logo from '@/assets/logo.webp'

import { useAppSelector } from '@/hooks/redux'
import { getToken } from '@/features/user/userSlice'
import { SignInForm } from '@/features/auth/components/SignInForm'
import { PageBox } from '@/components/PageBox/PageBox'

type LocationState = {
	from?: Location
}

export default function Auth() {
	const { palette } = useTheme()

	const navigate = useNavigate()
	const location = useLocation()

	const token = useAppSelector(getToken)

	useEffect(() => {
		const to: string = (location.state as LocationState)?.from?.pathname || '/'
		if (token) navigate(to, { replace: true })
	}, [token, navigate, location.state])

	return (
		<PageBox>
			<Box
				sx={{
					display: 'flex',
					justifyContent: 'center',
					alignItems: 'center',
					flexDirection: 'column',
					flexGrow: 1,
				}}
			>
				<Box
					sx={{
						height: 70,
						overflow: 'hidden',
						display: 'flex',
						alignItems: 'center',
						justifyContent: 'center',
						mb: 6,
						mt: -8,
						img: { height: '100%', width: 'auto' },
					}}
				>
					<img src={Logo} alt='logo' />
				</Box>
				<Box
					sx={{
						mx: 3,
						borderRadius: 4,
						py: 2.5,
						px: 3.75,
						width: { sm: 400, xs: '100%' },
						boxShadow: '2px 2px 8px 0px #3636362b',
						background: palette.background.paper,
					}}
				>
					<SignInForm />
				</Box>
			</Box>
		</PageBox>
	)
}
