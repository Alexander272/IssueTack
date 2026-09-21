import { type FC, useState } from 'react'
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogContentText,
	DialogTitle,
	Tab,
	Tabs,
	Typography,
} from '@mui/material'

import { useAppSelector } from '@/hooks/redux'
import { getCurrentCapabilities, getIsManager } from '@/features/user/userSlice'
import { SubscriptionsTab } from '../components/NotificationSettings'
import { DeadlineRemindersTab } from '../components/DeadlineReminderSettings'

type TabKey = 'subscriptions' | 'deadlines'

const TAB_LABELS: Record<TabKey, string> = {
	subscriptions: 'Подписки',
	deadlines: 'Сроки',
}

export const NotificationSettings: FC = () => {
	const isManager = useAppSelector(getIsManager)
	const { memberGroupIds } = useAppSelector(getCurrentCapabilities)
	const isMember = memberGroupIds.length > 0

	const showSubscriptions = isManager
	const showDeadlines = isMember

	const [activeTab, setActiveTab] = useState<TabKey>(showSubscriptions ? 'subscriptions' : 'deadlines')
	const [dirty, setDirty] = useState(false)
	const [pendingTab, setPendingTab] = useState<TabKey | null>(null)

	// Страховка на случай смены прав/реалма: активной может оказаться только доступная вкладка.
	const resolvedTab: TabKey =
		activeTab === 'subscriptions' && showSubscriptions
			? 'subscriptions'
			: activeTab === 'deadlines' && showDeadlines
				? 'deadlines'
				: showSubscriptions
					? 'subscriptions'
					: 'deadlines'

	const handleTabChange = (next: TabKey) => {
		if (next === resolvedTab) return
		if (dirty) {
			setPendingTab(next)
			return
		}
		setActiveTab(next)
	}

	const confirmSwitch = () => {
		if (pendingTab) setActiveTab(pendingTab)
		setPendingTab(null)
	}

	return (
		<Box sx={{ flexGrow: 1, overflow: 'auto' }}>
			<Box sx={{ px: 3, pt: 3 }}>
				<Typography variant='h5' sx={{ fontWeight: 700, color: '#1f2937' }}>
					Уведомления
				</Typography>
				<Typography variant='body2' sx={{ color: '#6b7280', display: { xs: 'none', sm: 'block' } }}>
					Управляйте подписками на события и напоминаниями о сроках
				</Typography>
			</Box>

			{showSubscriptions && showDeadlines && (
				<Tabs
					value={resolvedTab}
					onChange={(_, value: TabKey) => handleTabChange(value)}
					sx={{ px: 3, borderBottom: '1px solid #e5e7eb' }}
				>
					<Tab value='subscriptions' label='Подписки' sx={{ textTransform: 'none', fontWeight: 600 }} />
					<Tab value='deadlines' label='Сроки' sx={{ textTransform: 'none', fontWeight: 600 }} />
				</Tabs>
			)}

			<Box sx={{ p: 3 }}>
				{resolvedTab === 'subscriptions' ? (
					<SubscriptionsTab onDirtyChange={setDirty} />
				) : (
					<DeadlineRemindersTab onDirtyChange={setDirty} />
				)}
			</Box>

			<Dialog open={Boolean(pendingTab)} onClose={() => setPendingTab(null)}>
				<DialogTitle>Несохранённые изменения</DialogTitle>
				<DialogContent>
					<DialogContentText>
						Изменения на вкладке «{TAB_LABELS[resolvedTab]}» не сохранены. Перейти на вкладку «
						{TAB_LABELS[pendingTab ?? resolvedTab]}» и потерять их?
					</DialogContentText>
				</DialogContent>
				<DialogActions>
					<Button onClick={() => setPendingTab(null)} color='inherit' sx={{ textTransform: 'none' }}>
						Отмена
					</Button>
					<Button onClick={confirmSwitch} color='error' variant='contained' sx={{ textTransform: 'none' }}>
						Перейти
					</Button>
				</DialogActions>
			</Dialog>
		</Box>
	)
}