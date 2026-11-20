import type { ICategory, ICategoryModifier } from '../types/category'
import type { ICategoryGroup } from '../types/categoryGroup'

export type Block = {
	key: string
	sectionId: string | null
	section?: ICategoryGroup
	categories: ICategory[]
}

export type DragPayload = { kind: 'block'; key: string } | { kind: 'category'; key: string; categoryId: string }
export type DropTarget = DragPayload
export type DropPosition = 'before' | 'after'
export type DropIndicator = { key: string; position: DropPosition }

export const buildBlocks = (categories: ICategory[], groups: ICategoryGroup[]): Block[] => {
	const blocks: Block[] = []
	const sectionBlocks = new Map<string, Block>()

	categories.forEach(category => {
		const sectionId = category.categoryGroupId ?? null
		if (sectionId) {
			let block = sectionBlocks.get(sectionId)
			if (!block) {
				block = {
					key: `section:${sectionId}`,
					sectionId,
					section: groups.find(g => g.id === sectionId),
					categories: [],
				}
				sectionBlocks.set(sectionId, block)
				blocks.push(block)
			}
			block.categories.push(category)
			return
		}
		blocks.push({ key: `root:${category.id}`, sectionId: null, categories: [category] })
	})

	return blocks
}

export const flatten = (blocks: Block[]): string[] => blocks.flatMap(b => b.categories.map(c => c.id))

const insertAt = <T>(items: T[], from: number, toIndex: number): T[] => {
	const next = [...items]
	const [moved] = next.splice(from, 1)
	next.splice(Math.max(0, from < toIndex ? toIndex - 1 : toIndex), 0, moved)
	return next
}

export const moveBlock = (blocks: Block[], fromKey: string, toIndex: number): Block[] => {
	const from = blocks.findIndex(b => b.key === fromKey)
	return from === -1 ? blocks : insertAt(blocks, from, toIndex)
}

export const moveCategory = (blocks: Block[], key: string, categoryId: string, toIndex: number): Block[] =>
	blocks.map(block => {
		if (block.key !== key) return block
		const from = block.categories.findIndex(c => c.id === categoryId)
		return from === -1 ? block : { ...block, categories: insertAt(block.categories, from, toIndex) }
	})

export const canDrop = (payload: DragPayload, target: DropTarget): boolean =>
	payload.kind === 'block' ? target.kind === 'block' : target.kind === 'category' && target.key === payload.key

export const targetKey = (target: DropTarget): string =>
	target.kind === 'block' ? `block:${target.key}` : `category:${target.categoryId}`

export const buildModifiersMap = (categories: ICategory[], extra?: ICategoryModifier[]) => {
	const map = new Map<string, ICategoryModifier[]>()
	const push = (m: ICategoryModifier) => {
		const list = map.get(m.categoryId)
		if (!list) return void map.set(m.categoryId, [m])
		if (!list.some(x => x.id === m.id)) list.push(m)
	}
	categories.forEach(c => c.modifiers?.forEach(push))
	extra?.forEach(push)
	map.forEach(list => list.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name, 'ru')))
	return map
}
