import { type FC, useState } from 'react'
import { useFormContext, useWatch } from 'react-hook-form'
import { Box, Button, Chip, Menu, MenuItem, Typography } from '@mui/material'
import { BellIcon, PlusIcon, RotateCcwIcon } from 'lucide-mui'

import type { IDeadlineReminders } from '../../deadlineRemindersApiSlice'
import { DEFAULT_REMINDERS, REMINDER_OPTIONS, sortReminders } from './reminderOptions'
import { ReminderChip } from './ReminderChip'
import { NotificationSectionCard } from '../NotificationSettings/NotificationSectionCard'

type MenuState = { el: HTMLElement; index: number | null }

export const RemindersCard: FC = () => {
	const { control, setValue } = useFormContext<IDeadlineReminders>()
	const reminders = useWatch({ control, name: 'reminders' }) ?? []
	const sorted = sortReminders(reminders)

	const [menu, setMenu] = useState<MenuState | null>(null)
	const available = REMINDER_OPTIONS.filter(option => !sorted.includes(option.value))

	const editIndex = menu?.index ?? null
	const menuOptions =
		editIndex == null
			? available
			: REMINDER_OPTIONS.filter(option => option.value === sorted[editIndex] || !sorted.includes(option.value))

	const closeMenu = () => setMenu(null)

	const updateAt = (index: number, value: number) => {
		const next = [...sorted]
		next[index] = value
		setValue('reminders', sortReminders(next), { shouldDirty: true })
		closeMenu()
	}

	const removeAt = (index: number) => {
		setValue(
			'reminders',
			sorted.filter((_, i) => i !== index),
			{ shouldDirty: true },
		)
	}

	const handleSelect = (value: number) => {
		if (menu?.index == null) {
			setValue('reminders', sortReminders([...sorted, value]), { shouldDirty: true })
		} else {
			updateAt(menu.index, value)
			return
		}
		closeMenu()
	}

	return (
		<NotificationSectionCard
			title='Напоминания о сроках'
			subtitle='За сколько до дедлайна присылать напоминание'
			icon={BellIcon}
			actions={
				<Button
					color='inherit'
					onClick={() => setValue('reminders', sortReminders(DEFAULT_REMINDERS), { shouldDirty: true })}
					sx={{ textTransform: 'none' }}
				>
					<RotateCcwIcon sx={{ fontSize: 16, color: 'inherit', mr: 0.5 }} />
					Сбросить к значениям по умолчанию
				</Button>
			}
		>
			<Box sx={{ p: 3 }}>
				{sorted.length === 0 && (
					<Typography variant='body2' sx={{ color: '#6b7280', mb: 2 }}>
						Напоминания отключены — добавьте хотя бы одно.
					</Typography>
				)}

				<Box sx={{ display: 'flex', flexWrap: 'wrap', alignItems: 'center', gap: 1 }}>
					{sorted.map((value, index) => (
						<ReminderChip
							key={value}
							value={value}
							onClick={e => setMenu({ el: e.currentTarget, index })}
							onDelete={() => removeAt(index)}
							sx={{ cursor: 'pointer' }}
						/>
					))}

					<Chip
						variant='outlined'
						icon={<PlusIcon sx={{ fontSize: 16 }} />}
						label='Добавить'
						disabled={available.length === 0}
						onClick={e => setMenu({ el: e.currentTarget, index: null })}
						sx={{ borderStyle: 'dashed', pl: 0.5 }}
					/>
				</Box>

				<Menu anchorEl={menu?.el ?? null} open={Boolean(menu)} onClose={closeMenu}>
					{menuOptions.map(option => (
						<MenuItem key={option.value} onClick={() => handleSelect(option.value)}>
							<ReminderChip value={option.value} />
						</MenuItem>
					))}
				</Menu>
			</Box>
		</NotificationSectionCard>
	)
}
