import { type FC, useState } from 'react'
import { Box, Typography, Button } from '@mui/material'
import { PlayCircle } from 'lucide-mui'

import {
	useGetPeriodicTaskTemplatesQuery,
	useRunPeriodicTaskMutation,
	useGetPeriodicTaskInstancesQuery,
} from '../../equipmentApiSlice'
import type { IEquipment, IEquipmentPeriodicTaskInstance } from '../../types/equipment'
import { getShortDate, getSmartDate } from '@/utils/date'
import { periodLabel } from '../../utils/period'
import { Switch } from '@/components/Switch/Switch'

type Props = {
	unit: IEquipment
}

export const PeriodicTasksTab: FC<Props> = ({ unit }) => {
	// shiftByTemplate — ручной запуск: по умолчанию все шаблоны сдвигают следующий старт.
	const [shiftByTemplate, setShiftByTemplate] = useState<Record<string, boolean>>({})
	const getShiftValue = (templateId: string) =>
		shiftByTemplate[templateId] !== undefined ? shiftByTemplate[templateId] : true

	const { data: templatesResponse } = useGetPeriodicTaskTemplatesQuery(
		{ realmId: unit.realmId, equipmentTypeId: unit.typeId ?? undefined },
		{ skip: !unit.typeId },
	)
	const safeTemplates = Array.isArray(templatesResponse?.data) ? templatesResponse.data : []

	const { data: instancesResponse } = useGetPeriodicTaskInstancesQuery({
		realmId: unit.realmId,
		equipmentId: unit.id,
		status: 'pending',
		limit: 100,
	})
	// pending ищется по шаблону: у каждого шаблона свой ближайший запуск, поэтому
	// глобальный find() показал бы расписание только одному из них.
	const pendingByTemplate = new Map<string, IEquipmentPeriodicTaskInstance>()
	for (const instance of instancesResponse?.data ?? []) {
		if (instance.status === 'pending' && !pendingByTemplate.has(instance.templateId)) {
			pendingByTemplate.set(instance.templateId, instance)
		}
	}

	const [runTask] = useRunPeriodicTaskMutation()

	const handleRun = async (templateId: string) => {
		try {
			const dto = {
				realmId: unit.realmId,
				templateId,
				equipmentId: unit.id,
				shiftNext: getShiftValue(templateId),
			}
			await runTask(dto).unwrap()
		} catch {
			// Ошибка уже показана в тосте хуком
		}
	}

	return (
		<Box>
			{safeTemplates.length === 0 ? (
				<Box
					sx={{
						p: 4,
						border: '1px solid #e5e7eb',
						borderRadius: '8px',
						textAlign: 'center',
						color: '#6b7280',
					}}
				>
					<Typography variant='body2'>Нет шаблонов для периодических задач</Typography>
				</Box>
			) : (
				<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
					{safeTemplates.map(template => {
						const scheduledAt = pendingByTemplate.get(template.id)?.scheduledAt ?? null
						const isOverdue = scheduledAt ? new Date(scheduledAt) < new Date() : false

						return (
							<Box
								key={template.id}
								sx={{
									py: 1.5,
									px: 2,
									border: '1px solid #e5e7eb',
									borderRadius: '8px',
									transition: 'border-color 0.15s ease-in-out',
									'&:hover': { borderColor: '#93c5fd' },
									bgcolor: '#fff',
								}}
							>
								<Box sx={{ display: 'flex', gap: 3, alignItems: 'flex-start' }}>
									{/* Иконка */}
									<Box
										sx={{
											width: 40,
											height: 40,
											borderRadius: '8px',
											bgcolor: '#fffbeb',
											display: 'flex',
											alignItems: 'center',
											justifyContent: 'center',
											flexShrink: 0,
										}}
									>
										<PlayCircle sx={{ fontSize: 20, color: '#d97706' }} />
									</Box>

									{/* Контент */}
									<Box sx={{ flex: 1, minWidth: 0 }}>
										<Box
											sx={{
												display: 'flex',
												alignItems: 'center',
												gap: 2,
												flexWrap: 'wrap',
											}}
										>
											<Typography variant='subtitle2' sx={{ fontWeight: 600, color: '#111827' }}>
												{template.title}
											</Typography>
											{scheduledAt && (
												<Box
													sx={{
														display: 'inline-flex',
														alignItems: 'center',
														gap: 1,
														px: 2,
														py: 0.5,
														borderRadius: '9999px',
														fontSize: '11px',
														fontWeight: 500,
														bgcolor: isOverdue ? '#fee2e2' : '#eff6ff',
														color: isOverdue ? '#991b1b' : '#1e40af',
														border: `1px solid ${isOverdue ? '#fecaca' : '#bfdbfe'}`,
													}}
												>
													<PlayCircle sx={{ fontSize: 10 }} />
													{isOverdue ? 'Просрочено' : getSmartDate(scheduledAt!)}
												</Box>
											)}
										</Box>

										<Typography
											variant='caption'
											sx={{ color: '#6b7280', mb: 1, display: 'block' }}
										>
											Период:{' '}
											<Typography component='span' sx={{ fontWeight: 500, color: '#374151' }}>
												{periodLabel(template)}
											</Typography>
										</Typography>

										{/* Метрики */}
										<Box
											sx={{
												display: 'grid',
												gridTemplateColumns: 'repeat(2, 1fr)',
												gap: 2,
											}}
										>
											<Box
												sx={{
													bgcolor: '#f9fafb',
													borderRadius: '6px',
													px: 2,
													py: 1,
												}}
											>
												<Typography
													variant='caption'
													sx={{ color: '#6b7280', display: 'block' }}
												>
													Следующий запуск
												</Typography>
												<Typography
													variant='subtitle2'
													sx={{ fontWeight: 600, color: '#111827' }}
												>
													{scheduledAt ? getShortDate(scheduledAt) : 'Не запланирован'}
												</Typography>
											</Box>
											<Box
												sx={{
													bgcolor: '#f9fafb',
													borderRadius: '6px',
													px: 2,
													py: 1,
												}}
											>
												<Typography
													variant='caption'
													sx={{ color: '#6b7280', display: 'block' }}
												>
													Автостарт
												</Typography>
												<Typography
													variant='subtitle2'
													sx={{ fontWeight: 600, color: '#111827' }}
												>
													{template.autoStartOnFirstAssign ? 'От даты выдачи' : 'Вручную'}
												</Typography>
											</Box>
										</Box>

										{template.description && (
											<Typography
												variant='body2'
												sx={{
													color: '#6b7280',
													lineHeight: 1.5,
													mt: 1.5,
												}}
											>
												{template.description}
											</Typography>
										)}
									</Box>

									{/* Действия */}
									<Box
										sx={{
											display: 'flex',
											flexDirection: 'column',
											gap: 1.5,
											flexShrink: 0,
											minWidth: '160px',
											alignSelf: 'center',
										}}
										onClick={e => e.stopPropagation()}
									>
										<Button
											variant='outlined'
											startIcon={<PlayCircle sx={{ fontSize: 14 }} />}
											onClick={() => handleRun(template.id)}
											sx={{
												height: 36,
												borderRadius: '8px',
												textTransform: 'none',
												fontSize: '13px',
												fontWeight: 500,
											}}
										>
											Запустить сейчас
										</Button>
										{scheduledAt && (
											<Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mt: 1 }}>
												<Switch
													checked={getShiftValue(template.id)}
													onChange={e =>
														setShiftByTemplate(prev => ({
															...prev,
															[template.id]: e.target.checked,
														}))
													}
												/>
												<Typography variant='caption' sx={{ color: '#6b7280' }}>
													Сдвинуть следующий старт
												</Typography>
											</Box>
										)}
									</Box>
								</Box>
							</Box>
						)
					})}
				</Box>
			)}
		</Box>
	)
}
