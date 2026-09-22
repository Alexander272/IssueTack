import { AppBar, IconButton, Stack, Toolbar } from '@mui/material'
import { Link } from 'react-router'
import Logo from '@/assets/logo.webp'

interface MobileHeaderProps {
	onMenuClick?: () => void
}

export const MobileHeader = ({ onMenuClick }: MobileHeaderProps) => {
	return (
		<AppBar
			position='relative'
			sx={{ borderRadius: 0, alignItems: 'center', zIndex: theme => theme.zIndex.drawer + 1, display: { xs: 'flex', md: 'none' } }}
		>
			<Toolbar sx={{ justifyContent: 'space-between', width: '100%', maxWidth: 'xl' }}>
				<IconButton
					onClick={onMenuClick}
					sx={{ display: { md: 'none' }, mr: 1, color: 'text.primary', fontSize: '1.5rem' }}
				>
					☰
				</IconButton>

				<Stack
					component={Link}
					to='/'
					aria-label='home page'
					sx={{
						height: 50,
						overflow: 'hidden',
						alignItems: 'center',
						justifyContent: 'center',
						img: { height: '100%', width: 'auto' },
						position: 'absolute',
						left: '50%',
						transform: 'translateX(-50%)',
					}}
				>
					<img src={Logo} alt='logo' />
				</Stack>
			</Toolbar>
		</AppBar>
	)
}