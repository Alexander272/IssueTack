import { Fragment, useMemo, type FC, type ReactNode } from 'react'
import {
	Box,
	Stack,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	Typography,
	useTheme,
} from '@mui/material'
import { LayersIcon, UsersIcon } from 'lucide-mui'
import { useFormContext, useWatch } from 'react-hook-form'

import type { IGroup } from '@/features/groups/types/group'
import type { ICategory } from '@/features/categories/types/category'
import type { ICategoryNotificationSetting, INotificationSettings } from '../../notificationsApiSlice'
import { CATEGORY_EVENTS, type CategoryEventKey } from './constants'
import { useGetAllCategoriesQuery } from '@/features/categories/categoriesApiSlice'
import { useGetAllGroupsQuery } from '@/features/groups/groupsApiSlice'
import { BadgeCheckbox } from '@/components/BadgeCheckbox/BadgeCheckbox'
import { BoxFallback } from '@/components/Fallback/BoxFallback'
import { NotificationSectionCard } from './NotificationSectionCard'
import { SelectButtons } from './SelectButtons'

const EMPTY_SETTING = { newTask: false, status: false, comment: false, overdue: false }

export interface IGroupWithCategories extends IGroup {
	categories: ICategory[]
}

export const CategoriesCard: FC = () => {
	const { palette } = useTheme()
	const { data: categories, isFetching: isCategoriesFetching } = useGetAllCategoriesQuery()
	const { data: groups, isFetching: isGroupsFetching } = useGetAllGroupsQuery()

	const { setValue } = useFormContext<INotificationSettings>()
	const settings: ICategoryNotificationSetting[] = useWatch<INotificationSettings, 'categories'>({
		name: 'categories',
	})
	const enabled = useWatch<INotificationSettings, 'enabled'>({ name: 'enabled' })
	const disabled = !enabled

	const groupsWithCategories = useMemo<IGroupWithCategories[]>(() => {
		if (!groups?.data.length) return []

		return groups?.data.map(group => ({
			...group,
			categories: categories?.data.filter(category => category.groupId === group.id) || [],
		}))
	}, [groups, categories])

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

	const setAllGroup = (value: boolean, groupId: string) => {
		setValue(
			'categories',
			(categories?.data ?? [])
				.filter(c => c.groupId === groupId)
				.map(c => ({
					...categorySetting(c.id),
					newTask: value,
					status: value,
					comment: value,
					overdue: value,
				})),
			{ shouldDirty: true },
		)
	}

	const groupCountLabel = (count: number): string => {
		const mod10 = count % 10
		const mod100 = count % 100
		if (mod10 === 1 && mod100 !== 11) return `${count} категория`
		if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${count} категории`
		return `${count} категорий`
	}

	const renderEmpty: ReactNode = (
		<TableRow>
			<TableCell colSpan={CATEGORY_EVENTS.length + 1} align='center'>
				<Typography color={palette.text.secondary}>Категории не найдены</Typography>
			</TableCell>
		</TableRow>
	)

	if (isCategoriesFetching || isGroupsFetching) return <BoxFallback />
	return (
		<NotificationSectionCard
			title='Подписки по категориям'
			subtitle='Каждая категория привязана к группе. Включайте уведомления по нужным категориям.'
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
						{groupsWithCategories
						.filter(group => group.categories.length > 0)
						.map(group => {
							return (
								<Fragment key={group.id}>
									<TableRow sx={{ bgcolor: '#fbfbfb' }}>
										<TableCell sx={{ py: 1.25 }}>
											<Stack direction='row' spacing={1.5} sx={{ alignItems: 'center' }}>
												<UsersIcon sx={{ fontSize: 18, color: '#1d4ed8' }} />
												<Stack>
													<Typography
														variant='body2'
														sx={{ fontWeight: 600, color: 'text.primary' }}
													>
														{group.name}
													</Typography>
													<Typography
														component='p'
														variant='caption'
														sx={{ color: '#6b7280' }}
													>
														{groupCountLabel(group.categories.length)}
													</Typography>
												</Stack>
											</Stack>
										</TableCell>
										<TableCell colSpan={4}>
											<SelectButtons
												disabled={disabled}
												onSetAll={value => setAllGroup(value, group.id)}
												sx={{ justifyContent: 'flex-end' }}
											/>
										</TableCell>
									</TableRow>

									{group.categories.map(cat => {
										const cs = categorySetting(cat.id)
										return (
											<TableRow key={cat.id} hover>
												<TableCell sx={{ py: 0.5, pl: 4 }}>
													<Stack direction='row' spacing={1.5} sx={{ alignItems: 'center' }}>
														<LayersIcon sx={{ fontSize: 18, color: '#1d4ed8' }} />
														<Stack>
															<Typography
																variant='body2'
																sx={{ fontWeight: 600, color: 'text.primary' }}
															>
																{cat.name}
															</Typography>
															<Typography
																component='p'
																variant='caption'
																sx={{ color: '#6b7280' }}
															>
																{cat.description}
															</Typography>
														</Stack>
													</Stack>
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
								</Fragment>
							)
						})}

						{!categories?.data.length && renderEmpty}
					</TableBody>
				</Table>
			</TableContainer>
		</NotificationSectionCard>
	)
}
