import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import { TableCell, TableRow, Typography, Box } from '@mui/material'
import { ChevronDownIcon } from 'lucide-mui'

import type { ITask } from '../../types/task'
import type { GroupByField } from '../../constants/taskMaps'
import { useAppSelector } from '@/hooks/redux'
import { getHasNoGroups } from '@/features/user/userSlice'
import { TaskRow } from './TaskRow'
import { getGroupValue, sortGroupKeys } from './getGroupValue'

interface Props {
	tasks: ITask[]
	groupBy: GroupByField
	onTaskClick: (task: ITask) => void
	columnsCount: number
}

const STORAGE_KEY = '@issueTrack/tasksCollapsedGroups'
type CollapsedState = Partial<Record<GroupByField, string[]>>

const getInitialCollapsed = (): CollapsedState => {
	try {
		const saved = localStorage.getItem(STORAGE_KEY)
		return saved ? JSON.parse(saved) : {}
	} catch {
		return {}
	}
}

// Мемоизируем строку задачи
const MemoizedTaskRow = memo(TaskRow)

export const TaskTableGrouped = ({ tasks, groupBy, onTaskClick, columnsCount }: Props) => {
	const [collapsedMap, setCollapsedMap] = useState<CollapsedState>(getInitialCollapsed)
	const hasNoGroups = useAppSelector(getHasNoGroups)

	// Дебаунс для записи в localStorage (предотвращает фризы при частых кликах)
	useEffect(() => {
		const timeoutId = setTimeout(() => {
			try {
				localStorage.setItem(STORAGE_KEY, JSON.stringify(collapsedMap))
			} catch (e) {
				// В продакшене лучше использовать тихий логгер (например, Sentry)
				if (process.env.NODE_ENV === 'development') {
					console.error('Не удалось сохранить состояние групп:', e)
				}
			}
		}, 300) // Задержка 300мс

		return () => clearTimeout(timeoutId)
	}, [collapsedMap])

	const collapsedGroups = useMemo(() => new Set(collapsedMap[groupBy] ?? []), [collapsedMap, groupBy])

	const toggleGroup = useCallback(
		(key: string) => {
			setCollapsedMap(prev => {
				const currentList = prev[groupBy] ?? []
				const isCurrentlyCollapsed = currentList.includes(key)

				const nextList = isCurrentlyCollapsed ? currentList.filter(item => item !== key) : [...currentList, key]

				return {
					...prev,
					[groupBy]: nextList,
				}
			})
		},
		[groupBy],
	)

	const { groups, sortedKeys } = useMemo(() => {
		const grouped: Record<string, ITask[]> = {}
		tasks.forEach(task => {
			const key = getGroupValue(task, groupBy, hasNoGroups)
			if (!grouped[key]) grouped[key] = []
			grouped[key].push(task)
		})

		return {
			groups: grouped,
			sortedKeys: sortGroupKeys(Object.keys(grouped), groupBy),
		}
	}, [tasks, groupBy, hasNoGroups])

	// Обработчик клавиш для доступности
	const handleKeyDown = useCallback(
		(e: React.KeyboardEvent, key: string) => {
			if (e.key === 'Enter' || e.key === ' ') {
				e.preventDefault() // Предотвращаем скролл страницы при нажатии Space
				toggleGroup(key)
			}
		},
		[toggleGroup],
	)

	return (
		<>
			{sortedKeys.flatMap(key => {
				const groupTasks = groups[key]
				const isCollapsed = collapsedGroups.has(key)
				const groupId = `group-content-${key}`

				return [
					<TableRow
						key={`group-row-${key}`}
						sx={{
							bgcolor: '#f9fafb',
							cursor: 'pointer',
							userSelect: 'none',
							'&:hover': { bgcolor: '#f3f4f6' },
							// Убираем outline при фокусе с клавиатуры, заменяя на MUI focus-visible если нужно
							'&:focus-visible': { outline: '2px solid #3b82f6' },
						}}
						onClick={() => toggleGroup(key)}
						onKeyDown={e => handleKeyDown(e, key)}
						// Улучшения доступности
						tabIndex={0}
						role='button'
						aria-expanded={!isCollapsed}
						aria-controls={groupId}
					>
						<TableCell colSpan={columnsCount} sx={{ px: 3, py: 2 }}>
							<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
								<ChevronDownIcon
									aria-hidden='true' // Скрываем иконку от скринридеров
									sx={{
										fontSize: 16, // 10px слишком мелко, лучше 16-20px
										color: '#6b7280',
										transform: isCollapsed ? 'rotate(-90deg)' : 'rotate(0deg)',
										transition: 'transform 0.2s ease',
										flexShrink: 0,
									}}
								/>
								<Typography
									sx={{
										fontWeight: 700,
										color: '#374151',
										fontSize: '0.75rem',
										textTransform: 'uppercase',
										letterSpacing: '0.05em',
									}}
								>
									{key}
								</Typography>
								<Typography sx={{ fontSize: '0.75rem', color: '#6b7280', fontWeight: 500 }}>
									({groupTasks.length})
								</Typography>
							</Box>
						</TableCell>
					</TableRow>,
					...(isCollapsed
						? []
						: groupTasks.map(task => (
								<MemoizedTaskRow key={task.id} task={task} onClick={onTaskClick} groupBy={groupBy} />
							))),
				]
			})}
		</>
	)
}
