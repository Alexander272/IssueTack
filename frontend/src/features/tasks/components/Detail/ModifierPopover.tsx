import { useMemo, useState, type FC } from 'react'
import { alpha, Box, Button, Divider, InputAdornment, Popover, Stack, TextField, Typography } from '@mui/material'
import { CheckIcon, ListTree, SearchIcon, TriangleAlertIcon } from 'lucide-mui'

import { useGetAllCategoryModifiersQuery } from '@/features/categories/categoriesApiSlice'
import type { ITask } from '../../types/task'
import type { ICategoryModifier } from '@/features/categories/types/category'
import { PRIORITY_MAP } from '../../constants/taskMaps'
import { pluralize } from '@/utils/plural'
import { useDebounce } from '@/hooks/useDebounce'
import { ConfirmDialog } from '@/components/Dialogs/ConfirmDialog'

type Props = {
	task: ITask
	onSelect: (id: string | null) => void
	/** Контролируемый режим: открытие/якорь задаёт родитель (меню «Ещё» в шапке). */
	open?: boolean
	anchorEl?: HTMLElement | null
	onClose?: () => void
	/** Скрыть собственную кнопку-триггер — поповер открывается извне. */
	hideTrigger?: boolean
}

/** Выбор подкатегории на уже созданной заявке: исполнитель или начальник области.
 * Поповер устроен как DeadlinePopover — один контрол рядом с действиями заявки,
 * без открытия формы редактирования (иначе исполнитель получил бы доступ к
 * чужим полям). Отправляется только id и categoryModifierId.
 * По умолчанию неконтролируемый (своя кнопка); при open/anchorEl/onClose открытие
 * целиком задаёт родитель — так тот же поповер открывается из меню «Ещё». */
export const ModifierPopover: FC<Props> = ({ task, onSelect, open, anchorEl, onClose, hideTrigger }) => {
	const [internalAnchorEl, setInternalAnchorEl] = useState<HTMLButtonElement | null>(null)
	const controlled = open !== undefined
	const [pendingModifierId, setPendingModifierId] = useState<string | null | undefined>(undefined)
	const [search, setSearch] = useState('')
	const { data } = useGetAllCategoryModifiersQuery()

	const debounced = useDebounce(search, 300)

	const modifiers = (data?.data ?? []).filter(m => m.isActive && m.categoryId === task.category.id)
	const currentId = task.categoryModifier?.id ?? null
	const current = modifiers.find(m => m.id === currentId)

	// Фильтрация при поиске
	const filteredModifiers = useMemo(() => {
		const q = debounced.trim().toLowerCase()
		if (!q) return modifiers
		return modifiers.filter(m => m.name.toLowerCase().includes(q))
	}, [modifiers, debounced])

	const showSearch = modifiers.length >= 6

	const effectiveAnchorEl: HTMLElement | null = controlled ? (anchorEl ?? null) : internalAnchorEl
	const close = () => {
		if (controlled) onClose?.()
		else setInternalAnchorEl(null)
	}
	const apply = (id: string | null) => {
		if (id === currentId) {
			close()
			return
		}
		// 2. Было пусто (null), выбрали значение — применяем сразу без подтверждения
		if (currentId === null) {
			close()
			onSelect(id)
			return
		}
		// 3. Было конкретное значение, меняем на другое или сбрасываем на null — нужен конфирм
		setPendingModifierId(id)
	}

	const confirmChange = () => {
		if (pendingModifierId === undefined) return

		const nextId = pendingModifierId
		setPendingModifierId(undefined)
		close()
		onSelect(nextId)
	}

	const pendingName = modifiers.find(m => m.id === pendingModifierId)?.name ?? 'выбранной подкатегории'

	// В неконтролируемом режиме нечего показывать, если у категории нет active
	// подкатегорий. В контролируемом — меню уже открыто, и «Без подкатегории»
	// остаётся единственным способом снять устаревшую/неактивную подкатегорию.
	if (!controlled && modifiers.length == 0) return null

	return (
		<>
			{!hideTrigger && (
				<Button
					variant='outlined'
					onClick={e => setInternalAnchorEl(e.currentTarget)}
					sx={{
						maxWidth: 220,
						textTransform: 'none',
						boxShadow: 'none',
						'&:hover': { boxShadow: 'none' },
					}}
					startIcon={<ListTree sx={{ fontSize: 16 }} />}
				>
					<Box component='span' sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
						{current ? current.name : 'Подкатегория'}
					</Box>
				</Button>
			)}

			<Popover
				open={controlled ? Boolean(open) : Boolean(internalAnchorEl)}
				anchorEl={effectiveAnchorEl}
				onClose={close}
				anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
				transformOrigin={{ vertical: 'top', horizontal: 'right' }}
				disableScrollLock
				slotProps={{
					paper: {
						sx: {
							width: 320,
							borderRadius: 3,
							border: '1px solid',
							borderColor: 'divider',
							boxShadow: '0 12px 32px rgba(0,0,0,0.1)',
							mt: 1,
						},
					},
				}}
			>
				<Box>
					{showSearch && (
						<Box sx={{ p: 1 }}>
							<TextField
								fullWidth
								autoFocus
								placeholder='Найти подкатегорию'
								value={search}
								onChange={e => setSearch(e.target.value)}
								slotProps={{
									input: {
										startAdornment: (
											<InputAdornment position='start'>
												<SearchIcon sx={{ fontSize: 14 }} />
											</InputAdornment>
										),
									},
								}}
							/>
						</Box>
					)}

					{filteredModifiers.length === 0 ? (
						<Typography
							variant='body2'
							sx={{ py: 2.5, px: 1.5, textAlign: 'center', color: 'text.secondary' }}
						>
							Ничего не найдено
						</Typography>
					) : null}

					<Box
						sx={{
							display: 'flex',
							flexWrap: 'wrap',
							gap: 1,
							p: 1,
							maxHeight: 320,
							overflowY: 'auto',
						}}
					>
						{filteredModifiers.map(m => (
							<ModifierItem key={m.id} currentId={currentId} mod={m} onSelect={() => apply(m.id)} />
						))}
					</Box>
					<Divider />
					<Box sx={{ p: 1 }}>
						<Button
							onClick={() => apply(null)}
							sx={{
								display: 'block',
								width: '100%',
								p: '9px 10px',
								border: 0,
								borderRadius: '8px',
								bgcolor: 'transparent',
								color: 'text.secondary',
								font: 'inherit',
								fontSize: 14,
								textAlign: 'left',
								cursor: 'pointer',
								textTransform: 'inherit',
								'&:hover': {
									bgcolor: 'action.hover',
									color: 'text.primary',
								},
							}}
						>
							Без подкатегории
						</Button>
					</Box>

					<Divider />
					<Stack direction={'row'} sx={{ py: 1, px: 1.5, background: '#fff7e1', color: '#463203' }}>
						<TriangleAlertIcon sx={{ fontSize: 16, color: '#463203', mt: 0.75, mr: 1.5 }} />
						<Typography variant='caption' sx={{ lineHeight: 1.25 }}>
							При выборе срок и приоритет подставятся из подкатегории
						</Typography>
					</Stack>
				</Box>
			</Popover>

			<ConfirmDialog
				open={pendingModifierId !== undefined}
				title='Изменить подкатегорию?'
				message={`Срок и приоритет будут заменены значениями «${pendingName}». Срок считается от даты создания заявки.`}
				confirmLabel='Изменить'
				onConfirm={confirmChange}
				onCancel={() => setPendingModifierId(undefined)}
			/>
		</>
	)
}

type ItemProps = {
	currentId: string | null
	mod: ICategoryModifier
	onSelect: (id: string) => void
}
export const ModifierItem: FC<ItemProps> = ({ currentId, mod, onSelect }) => {
	const isSelected = mod.id === currentId
	const dotColor = mod.priority ? PRIORITY_MAP[mod.priority] : undefined

	const priorityLabel = mod.priority ? PRIORITY_MAP[mod.priority]?.label : null
	const dueDaysLabel = mod.defaultDueDays
		? `${mod.defaultDueDays} ${pluralize(mod.defaultDueDays, ['день', 'дня', 'дней'])}`
		: null

	const metaText = [priorityLabel, dueDaysLabel].filter(Boolean).join(' · ')

	return (
		<Box
			role='option'
			onClick={() => onSelect(mod.id)}
			sx={theme => ({
				display: 'grid',
				gridTemplateColumns: '20px 1fr auto',
				gap: '10px',
				alignItems: 'center',
				width: '100%',
				p: '9px 10px',
				borderRadius: '8px',
				cursor: 'pointer',
				bgcolor: isSelected ? alpha(theme.palette.primary.main, 0.1) : 'transparent',
				'&:hover': {
					bgcolor: isSelected ? alpha(theme.palette.primary.main, 0.14) : 'action.hover',
				},
			})}
		>
			{/* Круглый чекбокс */}
			<Box
				sx={{
					width: 18,
					height: 18,
					borderRadius: '50%',
					border: '1.5px solid',
					borderColor: isSelected ? 'primary.main' : 'divider',
					bgcolor: isSelected ? 'primary.main' : 'transparent',
					display: 'grid',
					placeItems: 'center',
					transition: 'all 0.15s',
				}}
			>
				{isSelected && <CheckIcon sx={{ fontSize: 13, color: '#fff', strokeWidth: 2 }} />}
			</Box>

			{/* Название */}
			<Typography variant='body2' noWrap sx={{ color: isSelected ? 'primary.main' : 'text.primary' }}>
				{mod.name}
			</Typography>

			{/* Мета-информация: приоритет и срок */}
			{(mod.priority || mod.defaultDueDays) && (
				<Box
					sx={{
						display: 'flex',
						alignItems: 'center',
						gap: 1,
						color: 'text.secondary',
						fontSize: '12.5px',
						whiteSpace: 'nowrap',
					}}
				>
					{dotColor?.barColor ? (
						<Box
							sx={{
								width: 8,
								height: 8,
								borderRadius: '50%',
								bgcolor: dotColor.barColor,
							}}
						/>
					) : null}
					<span>{metaText}</span>
				</Box>
			)}
		</Box>
	)
}
