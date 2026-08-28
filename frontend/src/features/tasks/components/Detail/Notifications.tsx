import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import {
	useGetIsSubscribedQuery,
	useSubscribeMutation,
	useUnsubscribeMutation,
} from '../../tasksApiSlice'

interface Props {
	taskId: string
}

export const Notifications = ({ taskId }: Props) => {
	const { data, isLoading } = useGetIsSubscribedQuery(taskId)
	const [subscribe] = useSubscribeMutation()
	const [unsubscribe] = useUnsubscribeMutation()

	if (isLoading) return <BoxFallback />

	const subscribed = data?.data?.subscribed ?? false

	const handleChange = async (checked: boolean) => {
		if (checked) {
			await subscribe(taskId)
		} else {
			await unsubscribe(taskId)
		}
	}

	return (
		<Box sx={{ bgcolor: 'white', borderRadius: '12px', border: '1px solid #e5e7eb', overflow: 'hidden' }}>
			<Box sx={{ px: 2.5, py: 1.5, borderBottom: '1px solid #e5e7eb', bgcolor: '#f9fafb' }}>
				<Typography sx={{ fontSize: '0.75rem', fontWeight: 700, color: '#374151', textTransform: 'uppercase', letterSpacing: '0.05em' }}>
					Уведомления
				</Typography>
			</Box>
			<Box sx={{ p: 2.5 }}>
				<FormControlLabel
					control={
						<Checkbox
							checked={subscribed}
							onChange={e => handleChange(e.target.checked)}
							sx={{ '&.Mui-checked': { color: 'primary.main' } }}
						/>
					}
					label={<Typography sx={{ fontSize: '0.8125rem', color: '#374151' }}>Получать уведомления</Typography>}
				/>
				<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af', mt: 0.5 }}>
					Вы будете получать уведомления о всех изменениях по этой заявке
				</Typography>
			</Box>
		</Box>
	)
}
