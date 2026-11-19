import { useTheme } from '@mui/material'

import { ErrorState } from '@/components/ErrorState/ErrorState'

export default function NotFound() {
	const { palette } = useTheme()

	return <ErrorState status={404} heading='Страница не найдена' accentColor={palette.primary.main} />
}
