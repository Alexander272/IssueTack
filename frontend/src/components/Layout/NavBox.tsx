import { Box, styled } from '@mui/material'

export const NavBox = styled(Box)(() => ({
	width: 46,
	height: 46,
	display: 'flex',
	justifyContent: 'center',
	alignItems: 'center',
	cursor: 'pointer',
	borderRadius: 12,
	transition: '.3s all ease-in-out',

	':hover': {
		background: '#05287f0a',
	},
}))
