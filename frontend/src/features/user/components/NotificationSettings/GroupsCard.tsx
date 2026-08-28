import { type FC } from 'react'
import { Box, Button, Typography, useTheme } from '@mui/material'
import { UsersIcon } from 'lucide-mui'
import { useFormContext, useWatch } from 'react-hook-form'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import type { IGroupNotificationSetting, INotificationSettings } from '../../notificationsApiSlice'
import { BadgeCheckbox } from '@/components/BadgeCheckbox/BadgeCheckbox'

import { NotificationSectionCard } from './NotificationSectionCard'
import { GROUP_EVENTS, type GroupEventKey } from './constants'

const EMPTY_SETTING = { newTask: false, overdue: false }

export const GroupsCard: FC = () => {
	const { palette } = useTheme()
	const { data: groups } = useGetAllGroupsQuery()
	const { setValue } = useFormContext<INotificationSettings>()
	const settings: IGroupNotificationSetting[] = useWatch<INotificationSettings, 'groups'>({ name: 'groups' })
	const enabled = useWatch<INotificationSettings, 'enabled'>({ name: 'enabled' })
	const disabled = !enabled

	const groupSetting = (id: string): IGroupNotificationSetting =>
		settings.find(g => g.id === id) ?? { id, ...EMPTY_SETTING }

	const toggleGroupEvent = (id: string, event: GroupEventKey) => {
		const existing = settings.find(g => g.id === id)
		const next = existing
			? settings.map(g => (g.id === id ? { ...g, [event]: !g[event] } : g))
			: [...settings, { id, ...EMPTY_SETTING, [event]: true }]
		setValue('groups', next, { shouldDirty: true })
	}

	const setAllGroups = (value: boolean) => {
		setValue(
			'groups',
			(groups?.data ?? []).map(g => ({
				...groupSetting(g.id),
				newTask: value,
				overdue: value,
			})),
			{ shouldDirty: true },
		)
	}

	return (
		<NotificationSectionCard
			title='Подписки по группам'
			subtitle='Дополнительные уведомления для задач конкретных групп'
			icon={UsersIcon}
			actions={
				<Box sx={{ display: 'flex', gap: 1 }}>
					<Button size='small' disabled={disabled} onClick={() => setAllGroups(true)}>
						Включить всё
					</Button>
					<Button size='small' color='inherit' disabled={disabled} onClick={() => setAllGroups(false)}>
						Выключить всё
					</Button>
				</Box>
			}
		>
			<Box sx={{ p: 2 }}>
				{(groups?.data ?? []).map(group => {
					const gs = groupSetting(group.id)
					return (
						<Box
							key={group.id}
							sx={{
								py: 1,
								px: 1.5,
								mb: 1.5,
								bgcolor: '#f9fafb',
								borderRadius: '10px',
								border: '1px solid #eef0f3',
							}}
						>
							<Box
								sx={{
									display: 'flex',
									alignItems: 'center',
									justifyContent: 'space-between',
									flexWrap: 'wrap',
									gap: 2,
								}}
							>
								<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
									<UsersIcon sx={{ fontSize: 22, color: '#64748b' }} />
									<Box>
										<Typography variant='body1' sx={{ fontWeight: 600, color: 'text.primary' }}>
											{group.name}
										</Typography>
										<Typography variant='caption' color={palette.text.secondary}>
											{group.members?.length ?? 0} участников
										</Typography>
									</Box>
								</Box>
								<Box sx={{ display: 'flex', gap: 3 }}>
									{GROUP_EVENTS.map(e => (
										<Box key={e.key} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
											<BadgeCheckbox
												checked={gs[e.key]}
												disabled={disabled}
												onChange={() => toggleGroupEvent(group.id, e.key)}
												ariaLabel={`${group.name}: ${e.label}`}
											/>
											<Typography variant='body2'>{e.label}</Typography>
										</Box>
									))}
								</Box>
							</Box>
						</Box>
					)
				})}
				{!groups?.data.length && <Typography color={palette.text.secondary}>Группы не найдены</Typography>}
			</Box>
		</NotificationSectionCard>
	)
}
