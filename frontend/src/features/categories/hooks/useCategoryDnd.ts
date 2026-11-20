import { useCallback, useMemo, useState, type DragEvent } from 'react'

import {
	canDrop,
	flatten,
	moveBlock,
	moveCategory,
	targetKey,
	type Block,
	type DragPayload,
	type DropIndicator,
	type DropPosition,
	type DropTarget,
} from '../utils/blocks'

export type CategoryDndHandlers = {
	start: (event: DragEvent<HTMLElement>, payload: DragPayload, row: HTMLElement | null) => void
	over: (event: DragEvent<HTMLElement>, target: DropTarget) => void
	leave: (event: DragEvent<HTMLElement>) => void
	drop: (event: DragEvent<HTMLElement>, target: DropTarget) => void
	reset: () => void
}

const positionFromEvent = (event: DragEvent<HTMLElement>): DropPosition => {
	const rect = event.currentTarget.getBoundingClientRect()
	return event.clientY < rect.top + rect.height / 2 ? 'before' : 'after'
}

type Options = { blocks: Block[]; disabled: boolean; onCommit: (ids: string[]) => void }

export const useCategoryDnd = ({ blocks, disabled, onCommit }: Options) => {
	const [dragging, setDragging] = useState<DragPayload | null>(null)
	const [dropTarget, setDropTarget] = useState<DropIndicator | null>(null)

	const reset = useCallback(() => {
		setDragging(null)
		setDropTarget(null)
	}, [])

	const start = useCallback(
		(event: DragEvent<HTMLElement>, payload: DragPayload, row: HTMLElement | null) => {
			if (disabled) return event.preventDefault()
			event.dataTransfer.effectAllowed = 'move'
			event.dataTransfer.setData('text/plain', JSON.stringify(payload))
			if (row) event.dataTransfer.setDragImage(row, 12, 12)
			setDragging(payload)
		},
		[disabled],
	)

	const over = useCallback(
		(event: DragEvent<HTMLElement>, target: DropTarget) => {
			if (!dragging || !canDrop(dragging, target)) return
			event.preventDefault()
			event.dataTransfer.dropEffect = 'move'
			const key = targetKey(target)
			const position = positionFromEvent(event)
			// Не дёргаем setState, если индикатор не изменился — иначе dragover перерисовывает всё на каждый пиксель.
			setDropTarget(prev => (prev?.key === key && prev.position === position ? prev : { key, position }))
		},
		[dragging],
	)

	const leave = useCallback((event: DragEvent<HTMLElement>) => {
		if (event.currentTarget.contains(event.relatedTarget as Node | null)) return
		setDropTarget(null)
	}, [])

	const drop = useCallback(
		(event: DragEvent<HTMLElement>, target: DropTarget) => {
			event.preventDefault()
			if (dragging && canDrop(dragging, target)) {
				const offset = positionFromEvent(event) === 'after' ? 1 : 0
				if (dragging.kind === 'block' && target.kind === 'block') {
					const toIndex = blocks.findIndex(b => b.key === target.key) + offset
					onCommit(flatten(moveBlock(blocks, dragging.key, toIndex)))
				} else if (dragging.kind === 'category' && target.kind === 'category') {
					const block = blocks.find(b => b.key === target.key)
					const targetIndex = block?.categories.findIndex(c => c.id === target.categoryId) ?? -1
					if (block && targetIndex >= 0) {
						onCommit(flatten(moveCategory(blocks, target.key, dragging.categoryId, targetIndex + offset)))
					}
				}
			}
			reset()
		},
		[dragging, blocks, onCommit, reset],
	)

	const handlers = useMemo<CategoryDndHandlers>(
		() => ({ start, over, leave, drop, reset }),
		[start, over, leave, drop, reset],
	)

	return { dragging, dropTarget, handlers }
}
