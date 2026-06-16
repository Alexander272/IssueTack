import { emphasize, styled } from '@mui/material'
import { Link } from 'react-router'

export const BreadLink = styled(Link)(({ theme }) => ({
	color: 'inherit',
	textDecoration: 'none',
	paddingRight: theme.spacing(1.5),
	paddingLeft: theme.spacing(1.5),
	paddingTop: theme.spacing(0.5),
	paddingBottom: theme.spacing(0.5),
	marginRight: `-${theme.spacing(0.8)}`,
	marginLeft: `-${theme.spacing(0.8)}`,
	borderRadius: 4 * (+theme.shape.borderRadius || 1),
	transition: 'all .3s ease-in-out',

	'&:hover, &:focus': {
		backgroundColor: emphasize(theme.palette.grey[100], 0.06),
		color: theme.palette.text.primary,
	},
	'&:active': {
		boxShadow: theme.shadows[1],
		backgroundColor: emphasize(theme.palette.grey[100], 0.12),
	},
}))
