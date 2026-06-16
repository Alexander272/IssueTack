import { createTheme } from '@mui/material/styles'

// import { type IScrollbarParameters, generateScrollbarStyles } from '@/utils/generateScrollbarStyles'

declare module '@mui/material/styles' {
	interface Palette {
		rowActive: Palette['primary']
	}

	interface PaletteOptions {
		rowActive?: PaletteOptions['primary']
	}
}

// const scrollbarParameters: IScrollbarParameters = {
// 	borderRadius: '5px',
// 	scrollbarBgColor: '#f2f2f2',
// 	scrollbarHeight: '.5rem',
// 	scrollbarWidth: '.5rem',
// 	thumbColor: '#00000020',
// 	thumbColorActive: '#00000050',
// 	thumbColorHover: '#00000030',
// }

export const theme = createTheme({
	palette: {
		primary: {
			main: '#0432a5',
			light: '#eaeefc',
		},
		background: {
			default: '#fafafa',
			paper: '#FFF',
		},
		rowActive: {
			main: '#c6d6ff',
			light: '#dde6fd',
		},
	},
	components: {
		// MuiCssBaseline: {
		// 	styleOverrides: generateScrollbarStyles(scrollbarParameters),
		// },
		MuiAppBar: {
			styleOverrides: {
				root: {
					border: 'none',
					boxShadow: '0 0 3px #0000004f',
					background: '#fff',
				},
			},
			defaultProps: {
				elevation: 0,
			},
		},
		MuiTooltip: {
			styleOverrides: {
				tooltip: {
					fontSize: '0.9rem',
					backgroundColor: '#000000de',
				},
				arrow: {
					color: '#000000de',
				},
			},
		},
		MuiPaper: {
			styleOverrides: {
				root: {
					borderRadius: 8,
				},
			},
		},
		MuiSelect: {
			defaultProps: {
				size: 'small',
			},
			styleOverrides: {
				root: {
					borderRadius: 12,
					'& fieldset': {
						transition: 'all 0.3s ease-in-out',
					},
				},
			},
		},
		MuiInputLabel: {
			defaultProps: {
				size: 'small',
			},
			// styleOverrides: {
			// 	root: {
			// 		transform: 'translate(14px, 6px) scale(1)',
			// 		'&.Mui-focused': {
			// 			transform: 'translate(14px, -9px) scale(0.75)',
			// 		},
			// 		'&.MuiInputLabel-shrink': {
			// 			transform: 'translate(14px, -9px) scale(0.75)',
			// 		},
			// 	},
			// },
		},
		MuiTextField: {
			defaultProps: {
				size: 'small',
				autoComplete: 'off',
				onWheel: event => (event.target as HTMLInputElement).blur(),
			},
			styleOverrides: {
				root: {
					'& fieldset': {
						transition: 'all 0.3s ease-in-out',
					},
				},
			},
		},
		MuiOutlinedInput: {
			// defaultProps: {
			// 	inputProps: {
			// 		step: 0.001,
			// 	},
			// },
			styleOverrides: {
				root: {
					borderRadius: 12,
				},
				// input: {
				// 	paddingTop: '6px',
				// 	paddingBottom: '6px',
				// 	'input[type=number]::-webkit-outer-spin-button': { WebkitAppearance: 'none', margin: 0 },
				// 	'input[type=number]::-webkit-inner-spin-button': { WebkitAppearance: 'none', margin: 0 },
				// 	'input[type=number]': { MozAppearance: 'textfield' },
				// },
			},
		},
		MuiInputBase: {
			defaultProps: {
				disableInjectingGlobalStyles: true,
			},
		},
		MuiButton: {
			styleOverrides: {
				root: {
					borderRadius: 12,
				},
			},
		},
		MuiBadge: {
			defaultProps: {
				anchorOrigin: {
					vertical: 'top',
					horizontal: 'left',
				},
			},
		},
	},
})
