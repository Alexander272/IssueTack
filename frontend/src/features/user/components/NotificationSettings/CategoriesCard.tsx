import { type FC, type ReactNode } from 'react'
import {
	Box,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
	useTheme,
} from '@mui/material'
import { LayersIcon } from 'lucide-mui'
import { useFormContext, useWatch } from 'react-hook-form'

import type { ICategoryNotificationSetting, INotificationSettings } from '../../notificationsApiSlice'
import { CATEGORY_EVENTS, type CategoryEventKey } from './constants'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { BadgeCheckbox } from '@/components/BadgeCheckbox/BadgeCheckbox'
import { NotificationSectionCard } from './NotificationSectionCard'
import { SelectButtons } from './SelectButtons'

const EMPTY_SETTING = { newTask: false, status: false, comment: false, overdue: false }

export const CategoriesCard: FC = () => {
	const { palette } = useTheme()
	const { data: categories } = useGetAllCategoriesQuery()
	const { setValue } = useFormContext<INotificationSettings>()
	const settings: ICategoryNotificationSetting[] = useWatch<INotificationSettings, 'categories'>({
		name: 'categories',
	})
	const enabled = useWatch<INotificationSettings, 'enabled'>({ name: 'enabled' })
	const disabled = !enabled

	const categorySetting = (id: string): ICategoryNotificationSetting =>
		settings.find(c => c.id === id) ?? { id, ...EMPTY_SETTING }

	const toggleCategoryEvent = (id: string, event: CategoryEventKey) => {
		const existing = settings.find(c => c.id === id)
		const next = existing
			? settings.map(c => (c.id === id ? { ...c, [event]: !c[event] } : c))
			: [...settings, { id, ...EMPTY_SETTING, [event]: true }]
		setValue('categories', next, { shouldDirty: true })
	}

	const setAllCategories = (value: boolean) => {
		setValue(
			'categories',
			(categories?.data ?? []).map(c => ({
				...categorySetting(c.id),
				newTask: value,
				status: value,
				comment: value,
				overdue: value,
			})),
			{ shouldDirty: true },
		)
	}

	const renderEmpty: ReactNode = (
		<TableRow>
			<TableCell colSpan={CATEGORY_EVENTS.length + 1} align='center'>
				<Typography color={palette.text.secondary}>Категории не найдены</Typography>
			</TableCell>
		</TableRow>
	)

	return (
		<NotificationSectionCard
			title='Подписки по категориям'
			subtitle='Уведомления по задачам выбранной категории во всех группах'
			icon={LayersIcon}
			actions={<SelectButtons disabled={disabled} onSetAll={setAllCategories} />}
		>
			<TableContainer>
				<Table sx={{ minWidth: 640 }}>
					<TableHead>
						<TableRow sx={{ bgcolor: '#f9fafb' }}>
							<TableCell sx={{ width: 260, fontWeight: 600 }}>Категория</TableCell>
							{CATEGORY_EVENTS.map(e => (
								<TableCell key={e.key} align='center' sx={{ fontWeight: 600 }}>
									<Box
										sx={{
											display: 'flex',
											justifyContent: 'center',
											alignItems: 'center',
											gap: 0.5,
										}}
									>
										<e.Icon sx={{ fontSize: 18, color: e.color, mr: 0.8 }} />
										<Typography variant='body2'>{e.label}</Typography>
									</Box>
								</TableCell>
							))}
						</TableRow>
					</TableHead>
					<TableBody>
						{(categories?.data ?? []).map(cat => {
							const cs = categorySetting(cat.id)
							return (
								<TableRow key={cat.id} hover>
									<TableCell sx={{ py: 0.5 }}>
										<Typography variant='body2' sx={{ fontWeight: 600, color: 'text.primary' }}>
											{cat.name}
										</Typography>
										<Typography variant='caption' color={palette.text.secondary}>
											{cat.description}
										</Typography>
									</TableCell>
									{CATEGORY_EVENTS.map(e => (
										<TableCell key={e.key} align='center'>
											<BadgeCheckbox
												checked={cs[e.key]}
												disabled={disabled}
												onChange={() => toggleCategoryEvent(cat.id, e.key)}
												ariaLabel={`${cat.name}: ${e.label}`}
											/>
										</TableCell>
									))}
								</TableRow>
							)
						})}
						{!categories?.data.length && renderEmpty}
					</TableBody>
				</Table>
			</TableContainer>
		</NotificationSectionCard>
	)
}
