import type { ICategory, ICategoryModifier } from '../types/category'
import type { ICategoryGroup } from '../types/categoryGroup'
import type { CategoryFilterState } from '../components/CategoryFilters'
import type { Block } from './blocks'
import { isCategoryHidden } from './visibility'

export type VisibleBlock = { block: Block; catVis: Map<string, boolean>; modVis: Map<string, boolean> }
export type TableVisibility = { blocks: VisibleBlock[]; emptySections: ICategoryGroup[] }

export const hasAttrFilters = (f: CategoryFilterState) =>
	f.group !== 'all' || f.status !== 'all' || f.priority !== 'all'

export const isFilteringActive = (f: CategoryFilterState, search: string) => Boolean(search.trim()) || hasAttrFilters(f)

type Params = {
	blocks: Block[]
	categoryGroups: ICategoryGroup[]
	modifiersByCategory: Map<string, ICategoryModifier[]>
	groupsMap: Map<string, string>
	filters: CategoryFilterState
	search: string
	showHidden: boolean
}

export const computeVisibility = ({
	blocks,
	categoryGroups,
	modifiersByCategory,
	groupsMap,
	filters,
	search,
	showHidden,
}: Params): TableVisibility => {
	const q = search.trim().toLowerCase()
	const attr = hasAttrFilters(filters)
	const filtering = Boolean(q) || attr
	const matches = (text: string) => !q || text.toLowerCase().includes(q)

	const catSelfMatch = (c: ICategory): boolean => {
		if (!matches([c.name, c.description || '', groupsMap.get(c.groupId) || ''].join(' '))) return false
		if (filters.group !== 'all' && c.groupId !== filters.group) return false
		if (filters.status === 'active' && !c.isActive) return false
		if (filters.status === 'inactive' && c.isActive) return false
		if (filters.priority !== 'all' && c.priority !== filters.priority) return false
		return true
	}

	const modSelfMatch = (m: ICategoryModifier, cat: ICategory): boolean => {
		if (!matches([m.name, m.description || '', groupsMap.get(cat.groupId) || ''].join(' '))) return false
		if (filters.group !== 'all' && cat.groupId !== filters.group) return false
		if (filters.status === 'active' && !m.isActive) return false
		if (filters.status === 'inactive' && m.isActive) return false
		if (filters.priority !== 'all' && (m.priority || cat.priority) !== filters.priority) return false
		return true
	}

	const visible: VisibleBlock[] = []

	blocks.forEach(block => {
		const section = block.section
		const secHidden = Boolean(section?.isHidden)
		const blockHidden = Boolean(block.sectionId && secHidden && !showHidden)
		const secSelf = block.sectionId && section ? matches(section.name) && !attr : false

		const catVis = new Map<string, boolean>()
		const modVis = new Map<string, boolean>()
		let anyCat = false

		block.categories.forEach(cat => {
			if ((isCategoryHidden(cat) || secHidden) && !showHidden) {
				catVis.set(cat.id, false)
				;(modifiersByCategory.get(cat.id) ?? []).forEach(m => modVis.set(m.id, false))
				return
			}

			const catSelf = catSelfMatch(cat)
			// Текстовый поиск по родителю (без атрибутных фильтров) показывает детей.
			const pm = (secSelf || (catSelf && Boolean(q))) && !attr
			let anyMod = false

			;(modifiersByCategory.get(cat.id) ?? []).forEach(m => {
				if (!m.isActive && !showHidden) return void modVis.set(m.id, false)
				const v = modSelfMatch(m, cat) || pm
				modVis.set(m.id, v)
				if (v) anyMod = true
			})

			const v = catSelf || pm || anyMod
			catVis.set(cat.id, v)
			if (v) anyCat = true
		})

		if (!blockHidden && (secSelf || anyCat)) visible.push({ block, catVis, modVis })
	})

	const withCategories = new Set(blocks.map(b => b.sectionId).filter(Boolean))
	const emptySections = categoryGroups.filter(g => {
		if (withCategories.has(g.id)) return false
		if (g.isHidden && !showHidden) return false
		return !filtering || (matches(g.name) && !attr)
	})

	return { blocks: visible, emptySections }
}
