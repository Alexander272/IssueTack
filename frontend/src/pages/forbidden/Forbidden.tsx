import { useTheme } from '@mui/material'

import { ErrorState } from '@/components/ErrorState/ErrorState'

export default function Forbidden() {
	const { palette } = useTheme()

	return <ErrorState status={403} heading='Доступ к данному разделу запрещен' accentColor={palette.primary.main} />
}
