import { useMemo, useState, type DragEvent, type FC } from 'react'
import { Box, Button, IconButton, Stack, Tooltip, Typography } from '@mui/material'
import { EditIcon, EyeOffIcon, GripVerticalIcon, LayersIcon, RotateCcwIcon, SaveIcon, TagIcon } from 'lucide-mui'
import { toast } from 'react-toastify'

import type { ICategory } from '../types/category'
import type { ICategoryGroup } from '@/features/categoryGroups/types/categoryGroup'
import { useReorderCategoriesMutation } from '../categoriesApiSlice'

// Блок — раздел категорий либо одиночная категория без раздела. Порядок блоков
// и категорий внутри них — это глобальный порядок категорий реалма.
type Block = {
	key: string
	sectionId: string | null
	name: string
	isHidden: boolean
	categories: ICategory[]
}

// Что именно тащит пользователь.
type DragPayload = { kind: 'block'; key: string } | { kind: 'category'; key: string; categoryId: string }

// На что можно бросить.
type DropTarget = { kind: 'block'; key: string } | { kind: 'category'; key: string; categoryId: string }

type Props = {
	categories: ICategory[]
	categoryGroups: ICategoryGroup[]
	onEditSection?: (id: string) => void
}

// buildBlocks собирает блоки из категорий, уже упорядоченных глобально:
// блок раздела начинается с первой его категории. Категория без раздела —
// отдельный блок из одной категории, поэтому несколько таких блоков могут
// стоять в разных местах между разделами.
const buildBlocks = (categories: ICategory[], groups: ICategoryGroup[]): Block[] => {
	const blocks: Block[] = []
	const sectionBlocks = new Map<string, Block>()

	categories.forEach(category => {
		const sectionId = category.categoryGroupId ?? null
		if (sectionId) {
			let block = sectionBlocks.get(sectionId)
			if (!block) {
				const group = groups.find(g => g.id === sectionId)
				block = {
					key: `section:${sectionId}`,
					sectionId,
					name: group?.name ?? '',
					isHidden: group?.isHidden ?? false,
					categories: [],
				}
				sectionBlocks.set(sectionId, block)
				blocks.push(block)
			}
			block.categories.push(category)
			return
		}
		blocks.push({
			key: `root:${category.id}`,
			sectionId: null,
			name: category.name,
			isHidden: category.isHidden,
			categories: [category],
		})
	})

	return blocks
}

const flatten = (blocks: Block[]): string[] => blocks.flatMap(block => block.categories.map(category => category.id))

// Перенос блока целиком: категории раздела едут вместе с ним.
const moveBlock = (blocks: Block[], fromKey: string, toIndex: number): Block[] => {
	const from = blocks.findIndex(block => block.key === fromKey)
	if (from === -1) return blocks

	const next = [...blocks]
	const [moved] = next.splice(from, 1)
	const index = from < toIndex ? toIndex - 1 : toIndex
	next.splice(index, 0, moved)
	return next
}

// Перенос категории — только внутри своего блока: раздел меняется в карточке
// категории, а не перетаскиванием.
const moveCategory = (blocks: Block[], key: string, categoryId: string, toIndex: number): Block[] =>
	blocks.map(block => {
		if (block.key !== key) return block
		const from = block.categories.findIndex(category => category.id === categoryId)
		if (from === -1) return block

		const categories = [...block.categories]
		const [moved] = categories.splice(from, 1)
		const index = from < toIndex ? toIndex - 1 : toIndex
		categories.splice(index, 0, moved)
		return { ...block, categories }
	})

// Бросок блока — только на другой блок, бросок категории — только на строку
// её же блока.
const canDrop = (payload: DragPayload, target: DropTarget): boolean => {
	if (payload.kind === 'block') return target.kind === 'block'
	return target.kind === 'category' && target.key === payload.key
}

// Ключ строки-индикатора: у блока и у категории он свой, иначе подсветка
// «линии вставки» задевала бы все строки блока сразу.
const targetKey = (target: DropTarget): string =>
	target.kind === 'block' ? `block:${target.key}` : `category:${target.categoryId}`

export const CategoryOrderEditor: FC<Props> = ({ categories, categoryGroups, onEditSection }) => {
	const [reorder, { isLoading }] = useReorderCategoriesMutation()

	const [dragging, setDragging] = useState<DragPayload | null>(null)
	const [dropTarget, setDropTarget] = useState<{ key: string; position: 'before' | 'after' } | null>(null)
	// Локальный (ещё не сохранённый) порядок. null — берём серверный.
	const [local, setLocal] = useState<{ signature: string; ids: string[] } | null>(null)

	const byId = useMemo(() => new Map(categories.map(category => [category.id, category])), [categories])
	const serverIds = useMemo(() => categories.map(category => category.id), [categories])
	// Подпись серверного порядка: изменилась — значит локальные правки
	// (если они не сохранены) больше не актуальны.
	const signature = useMemo(() => serverIds.join(','), [serverIds])

	const ids = local && local.signature === signature ? local.ids : serverIds
	const ordered = useMemo(
		() => ids.map(id => byId.get(id)).filter((category): category is ICategory => Boolean(category)),
		[ids, byId],
	)
	const blocks = useMemo(() => buildBlocks(ordered, categoryGroups), [ordered, categoryGroups])

	const isDirty = ids.join(',') !== serverIds.join(',')
	const emptySections = categoryGroups.filter(group => !blocks.some(block => block.sectionId === group.id))

	const setBlocks = (next: Block[]) => setLocal({ signature, ids: flatten(next) })

	const resetDrag = () => {
		setDragging(null)
		setDropTarget(null)
	}

	const handleDragStart = (event: DragEvent<HTMLSpanElement>, payload: DragPayload, row: HTMLElement | null) => {
		event.dataTransfer.effectAllowed = 'move'
		event.dataTransfer.setData('text/plain', JSON.stringify(payload))
		// Перетаскиваем всю строку, а не только ручку.
		if (row) event.dataTransfer.setDragImage(row, 12, 12)
		setDragging(payload)
	}

	const handleDragOver = (event: DragEvent<HTMLDivElement>, target: DropTarget) => {
		if (!dragging || !canDrop(dragging, target)) return
		event.preventDefault()
		event.dataTransfer.dropEffect = 'move'
		const rect = event.currentTarget.getBoundingClientRect()
		const position = event.clientY < rect.top + rect.height / 2 ? 'before' : 'after'
		setDropTarget({ key: targetKey(target), position })
	}

	// Индикатор гасим, когда курсор ушёл из строки насовсем. Внутри строки
	// dragleave тоже прилетает (переезд по дочерним элементам) — такие
	// пропускаем, иначе индикатор мигает.
	const handleDragLeave = (event: DragEvent<HTMLDivElement>) => {
		if (event.currentTarget.contains(event.relatedTarget as Node | null)) return
		setDropTarget(null)
	}

	const handleDrop = (event: DragEvent<HTMLDivElement>, target: DropTarget) => {
		event.preventDefault()
		if (!dragging || !canDrop(dragging, target)) {
			resetDrag()
			return
		}

		const rect = event.currentTarget.getBoundingClientRect()
		const position = event.clientY < rect.top + rect.height / 2 ? 'before' : 'after'

		if (dragging.kind === 'block' && target.kind === 'block') {
			const toIndex = blocks.findIndex(block => block.key === target.key) + (position === 'after' ? 1 : 0)
			setBlocks(moveBlock(blocks, dragging.key, toIndex))
		}

		if (dragging.kind === 'category' && target.kind === 'category') {
			const block = blocks.find(item => item.key === target.key)
			const targetIndex = block?.categories.findIndex(category => category.id === target.categoryId) ?? -1
			if (block && targetIndex >= 0) {
				const toIndex = targetIndex + (position === 'after' ? 1 : 0)
				setBlocks(moveCategory(blocks, target.key, dragging.categoryId, toIndex))
			}
		}

		resetDrag()
	}

	const handleSave = async () => {
		try {
			await reorder({ categoryIds: ids }).unwrap()
			toast.success('Порядок категорий сохранён')
		} catch {
			toast.error('Не удалось сохранить порядок', { autoClose: false })
		}
	}

	const indicator = (key: string, position: 'before' | 'after') =>
		dropTarget?.key === key && dropTarget.position === position

	return (
		<Box>
			<Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 2, mb: 1.5 }}>
				<Typography sx={{ fontSize: '0.8rem', color: '#6b7280' }}>
					Перетащите раздел или категорию — порядок изменится в списках выбора. Раздел переносится вместе со своими категориями, категории без раздела можно переносить как отдельные блоки между разделами и другими одиночными категориями.
				</Typography>
				<Box sx={{ display: 'flex', gap: 1, flexShrink: 0 }}>
					<Tooltip title='Вернуть порядок как на сервере'>
						<span>
							<IconButton size='small' onClick={() => setLocal(null)} disabled={!isDirty || isLoading}>
								<RotateCcwIcon sx={{ fontSize: 18 }} />
							</IconButton>
						</span>
					</Tooltip>
					<Button
						variant='contained'
						size='small'
						onClick={handleSave}
						disabled={!isDirty || isLoading}
						sx={{ textTransform: 'none', borderRadius: '8px' }}
					>
						<SaveIcon sx={{ fontSize: 16, mr: 1 }} />
						Сохранить порядок
					</Button>
				</Box>
			</Box>

			<Stack spacing={0.5}>
				{blocks.map(block => (
					<Box
						key={block.key}
						sx={{
							borderRadius: '8px',
							borderTop: '2px solid',
							borderBottom: '2px solid',
							borderColor:
								indicator(`block:${block.key}`, 'before') || indicator(`block:${block.key}`, 'after')
									? '#2563eb'
									: 'transparent',
						}}
					>
						<Box
							onDragOver={event => handleDragOver(event, { kind: 'block', key: block.key })}
							onDragLeave={handleDragLeave}
							onDrop={event => handleDrop(event, { kind: 'block', key: block.key })}
							sx={{
								display: 'flex',
								alignItems: 'center',
								gap: 1,
								py: 0.75,
								px: 1,
								borderRadius: '8px',
								bgcolor: '#f9fafb',
								opacity: dragging?.kind === 'block' && dragging.key === block.key ? 0.5 : 1,
							}}
						>
							<Tooltip title={block.sectionId ? 'Перетащите, чтобы изменить порядок раздела' : 'Перетащите, чтобы изменить положение категории'}>
								<Box
									component='span'
									draggable
									onDragStart={event => handleDragStart(event, { kind: 'block', key: block.key }, event.currentTarget.parentElement)}
									onDragEnd={resetDrag}
									sx={{ display: 'flex', cursor: 'grab', color: '#9ca3af', '&:active': { cursor: 'grabbing' } }}
								>
									<GripVerticalIcon sx={{ fontSize: 18 }} />
								</Box>
							</Tooltip>
							{block.sectionId ? (
								<LayersIcon sx={{ fontSize: 16, color: '#6b7280' }} />
							) : (
								<TagIcon sx={{ fontSize: 16, color: '#6b7280' }} />
							)}
							<Typography sx={{ fontWeight: block.sectionId ? 600 : 400 }}>{block.name}</Typography>
							{block.sectionId && block.isHidden && (
								<Tooltip title='Скрытый раздел — не виден заказчикам, его категории скрыты'>
									<EyeOffIcon sx={{ fontSize: 16, color: '#b45309' }} />
								</Tooltip>
							)}
							{!block.sectionId && block.isHidden && (
								<Tooltip title='Скрытая категория — не видна заказчикам'>
									<EyeOffIcon sx={{ fontSize: 16, color: '#b45309' }} />
								</Tooltip>
							)}
							{block.sectionId && (
								<Typography sx={{ fontSize: '0.75rem', color: '#9ca3af' }}>
									{block.categories.length} кат.
								</Typography>
							)}
							{onEditSection && block.sectionId && (
								<Tooltip title='Переименовать раздел'>
									<IconButton size='small' onClick={() => onEditSection(block.sectionId!)} sx={{ ml: 'auto' }}>
										<EditIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
									</IconButton>
								</Tooltip>
							)}
						</Box>

						{block.sectionId && (
							<Stack spacing={0.25} sx={{ pl: 4, pt: 0.5, pb: 0.5 }}>
								{block.categories.map(category => (
									<Box
										key={category.id}
										onDragOver={event => handleDragOver(event, { kind: 'category', key: block.key, categoryId: category.id })}
										onDragLeave={handleDragLeave}
										onDrop={event => handleDrop(event, { kind: 'category', key: block.key, categoryId: category.id })}
										sx={{
											display: 'flex',
											alignItems: 'center',
											gap: 1,
											py: 0.5,
											px: 1,
											borderRadius: '6px',
											borderTop: '2px solid',
											borderBottom: '2px solid',
											borderColor:
												indicator(`category:${category.id}`, 'before') ||
												indicator(`category:${category.id}`, 'after')
													? '#2563eb'
													: 'transparent',
											opacity:
												dragging?.kind === 'category' && dragging.categoryId === category.id ? 0.5 : 1,
										}}
									>
										<Tooltip title='Перетащите, чтобы изменить порядок внутри раздела'>
											<Box
												component='span'
												draggable
												onDragStart={event =>
													handleDragStart(
														event,
														{ kind: 'category', key: block.key, categoryId: category.id },
														event.currentTarget.parentElement,
													)
												}
												onDragEnd={resetDrag}
												sx={{ display: 'flex', cursor: 'grab', color: '#cbd5e1', '&:active': { cursor: 'grabbing' } }}
											>
												<GripVerticalIcon sx={{ fontSize: 16 }} />
											</Box>
										</Tooltip>
										<Typography sx={{ fontSize: '0.85rem' }}>{category.name}</Typography>
										{category.isHidden && (
											<Tooltip title='Скрытая категория — не видна заказчикам'>
												<EyeOffIcon sx={{ fontSize: 14, color: '#b45309' }} />
											</Tooltip>
										)}
									</Box>
								))}
							</Stack>
						)}
					</Box>
				))}

				{emptySections.length > 0 && (
					<Box sx={{ mt: 1 }}>
						<Typography sx={{ fontSize: '0.8rem', color: '#6b7280', mb: 0.5 }}>
							Разделы без категорий — на порядок в списках выбора не влияют:
						</Typography>
						<Stack spacing={0.5}>
							{emptySections.map(section => (
								<Box
									key={section.id}
									sx={{
										display: 'flex',
										alignItems: 'center',
										gap: 1,
										py: 0.75,
										px: 1,
										borderRadius: '8px',
										bgcolor: '#f9fafb',
									}}
								>
									<LayersIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
									<Typography sx={{ fontWeight: 600, color: '#6b7280' }}>{section.name}</Typography>
									{section.isHidden && <EyeOffIcon sx={{ fontSize: 16, color: '#b45309' }} />}
									{onEditSection && (
										<Tooltip title='Переименовать раздел'>
											<IconButton size='small' onClick={() => onEditSection(section.id)} sx={{ ml: 'auto' }}>
												<EditIcon sx={{ fontSize: 16, color: '#9ca3af' }} />
											</IconButton>
										</Tooltip>
									)}
								</Box>
							))}
						</Stack>
					</Box>
				)}
			</Stack>
		</Box>
	)
}
