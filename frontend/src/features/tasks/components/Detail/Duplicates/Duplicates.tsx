import { useMemo, useState } from 'react'
import { Box, Button, Typography } from '@mui/material'
import { LinkIcon } from 'lucide-mui'
import dayjs from 'dayjs'

import type { IDuplicateRef, ITask } from '../../../types/task'
import { useGetTicketDuplicatesQuery } from '../../../tasksApiSlice'
import { useDuplicateUnlink } from '../../../hooks/useDuplicateUnlink'
import { UnlinkDuplicateDialog } from '../UnlinkDuplicateDialog'
import { DuplicateRow } from './Row'

interface Props {
	task: ITask
}

/**
 * Группа строк внутри карточки. Сейчас одна («Дубли»); когда появится
 * «Повторные обращения», достаточно добавить ещё одну запись в groups —
 * заголовок карточки и отрисовка секций уже считаются их количества.
 */
interface RelatedGroup {
	key: string
	title: string
	items: IDuplicateRef[]
}

const VISIBLE_STEP = 5

export const Duplicates = ({ task }: Props) => {
	const [expanded, setExpanded] = useState<Record<string, number>>({})
	const { target: unlinkTarget, setTarget: setUnlinkTarget, unlinking, confirm: handleUnlink } = useDuplicateUnlink()

	const isDuplicate = Boolean(task.duplicateOf)

	// Список дублей грузим и без права связывания: это read-информация о заявке.
	const { data: duplicatesData } = useGetTicketDuplicatesQuery(task.id, { skip: isDuplicate })

	const duplicates = duplicatesData?.data
	const groups = useMemo<RelatedGroup[]>(
		() => [
			{
				key: 'duplicates',
				title: 'Дубли',
				items: [...(duplicates ?? [])].sort(
					(a, b) => dayjs(b.createdAt).valueOf() - dayjs(a.createdAt).valueOf(),
				),
			},
		],
		[duplicates],
	)
	const total = groups.reduce((sum, group) => sum + group.items.length, 0)
	const cardTitle = groups.length === 1 ? groups[0].title : 'Связанные заявки'

	if (isDuplicate || total === 0) return null

	return (
		<>
			<Box
				sx={{
					bgcolor: 'background.paper',
					borderRadius: '12px',
					border: '1px solid',
					borderColor: 'divider',
					overflow: 'hidden',
				}}
			>
				<Box
					sx={{
						px: 2.5,
						py: 1.5,
						borderBottom: '1px solid #e5e7eb',
						bgcolor: '#f9fafb',
						display: 'flex',
						alignItems: 'center',
						gap: 1,
					}}
				>
					<Typography
						sx={{
							fontSize: '0.75rem',
							fontWeight: 700,
							color: '#374151',
							textTransform: 'uppercase',
							letterSpacing: '0.05em',
							display: 'flex',
							alignItems: 'center',
							gap: 0.75,
						}}
					>
						<LinkIcon sx={{ fontSize: 14 }} />
						{cardTitle}
					</Typography>
					{total > 0 && (
						<Box
							component='span'
							sx={{
								fontSize: 12,
								fontWeight: 600,
								color: 'text.secondary',
								bgcolor: 'action.hover',
								border: '1px solid',
								borderColor: 'divider',
								borderRadius: '10px',
								px: 0.9,
							}}
						>
							{total}
						</Box>
					)}
				</Box>

				{total === 0 ? (
					<Typography sx={{ px: 2.5, py: 1.5, fontSize: 13, color: 'text.secondary' }}>Дублей нет</Typography>
				) : (
					groups
						.filter(group => group.items.length > 0)
						.map(group => {
							const visible = expanded[group.key] ?? VISIBLE_STEP
							const shown = group.items.slice(0, visible)
							const hidden = group.items.length - shown.length
							return (
								<Box
									key={group.key}
									className='related-group'
									sx={{
										'& + .related-group': {
											borderTop: '1px solid',
											borderColor: 'divider',
											mt: 0.75,
										},
									}}
								>
									{groups.length > 1 && (
										<Typography
											sx={{
												px: 2.5,
												pt: 1.5,
												pb: 0.25,
												fontSize: 12,
												fontWeight: 600,
												color: 'text.secondary',
											}}
										>
											{group.title}
										</Typography>
									)}
									{shown.map(duplicate => (
										<DuplicateRow
											key={duplicate.id}
											duplicate={duplicate}
											canUnlink={duplicate.canUnlink}
											unlinking={unlinking}
											onUnlink={item =>
												setUnlinkTarget({ id: item.id, ticketNumber: item.ticketNumber })
											}
										/>
									))}
									{hidden > 0 && (
										<Box sx={{ px: 2, py: 1, borderTop: '1px solid', borderColor: 'divider' }}>
											<Button
												size='small'
												onClick={() =>
													setExpanded(prev => ({
														...prev,
														[group.key]: visible + VISIBLE_STEP,
													}))
												}
												sx={{ textTransform: 'none', fontSize: 12, fontWeight: 600, px: 1 }}
											>
												Показать ещё {hidden}
											</Button>
										</Box>
									)}
								</Box>
							)
						})
				)}
			</Box>

			<UnlinkDuplicateDialog
				target={unlinkTarget}
				loading={unlinking}
				onConfirm={handleUnlink}
				onCancel={() => setUnlinkTarget(null)}
			/>
		</>
	)
}
