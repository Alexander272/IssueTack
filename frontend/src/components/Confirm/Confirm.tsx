import { type FC, type JSX, type MouseEvent, useState } from 'react'
import { Box, Button, Popover, Stack, Typography, useTheme, type SxProps } from '@mui/material'
import { WarnIcon } from '@/components/Icons/WarnIcon'

type Props = {
	onClick: () => void
	width?: string
	iconColor?: string
	buttonComponent?: JSX.Element
	confirmTitle?: string
	confirmText: string
	buttonColor?: 'error' | 'inherit' | 'primary' | 'secondary' | 'success' | 'info' | 'warning'
	sx?: SxProps
}

export const Confirm: FC<Props> = ({
	width,
	onClick,
	iconColor,
	buttonComponent,
	confirmTitle,
	confirmText,
	buttonColor = 'error',
	sx,
}) => {
	const [anchor, setAnchor] = useState<Element | null>(null)

	const { palette } = useTheme()

	const handleOpen = (event: MouseEvent) => setAnchor(event.currentTarget)
	const handleClose = () => setAnchor(null)

	const confirmHandler = (event: MouseEvent) => {
		event.stopPropagation()
		handleClose()
		onClick()
	}

	const open = Boolean(anchor)
	return (
		<Box sx={{ width: width ? width : 'inherit', height: '100%', ...sx }}>
			<Box onClick={handleOpen}>{buttonComponent}</Box>

			<Popover
				open={open}
				anchorEl={anchor}
				onClose={handleClose}
				anchorOrigin={{
					vertical: 'bottom',
					horizontal: 'center',
				}}
				transformOrigin={{
					vertical: 'top',
					horizontal: 'center',
				}}
			>
				<Stack spacing={2} sx={{ px: 2, py: 1.2 }}>
					<Box>
						<Stack
							spacing={1}
							direction={'row'}
							sx={{ justifyContent: 'center', alignItems: 'center', mb: 1 }}
						>
							<WarnIcon fill={iconColor || palette.error.main} />
							<Typography align='center' sx={{ fontSize: '1.1rem', fontWeight: 'bold' }}>
								{confirmTitle || 'Удаление'}
							</Typography>
						</Stack>

						<Typography align='center' sx={{ maxWidth: 260 }}>
							{confirmText}
						</Typography>
					</Box>

					<Stack direction='row' spacing={2}>
						<Button onClick={confirmHandler} variant='contained' color={buttonColor} fullWidth>
							Да
						</Button>
						<Button onClick={handleClose} variant='outlined' fullWidth>
							Отмена
						</Button>
					</Stack>
				</Stack>
			</Popover>
		</Box>
	)
}
