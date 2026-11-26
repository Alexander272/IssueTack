// Секция 1 формы создания: категория, площадка и подкатегория с подсказкой
// авто-подбора. Категории группируются по разделам (нативные optgroup): категория
// без раздела — отдельный блок из одного пункта, без группы «Без раздела».

import React, {useMemo} from 'react'

import type {PluginCategory, PluginCategoryModifier, PluginModifierShort, PluginSite} from '../../types'
import type {FieldErrors} from './model'

interface CategoryAndSiteSectionProps {
	categories: PluginCategory[]
	sites: PluginSite[]
	userSiteId: string
	categoryId: string
	siteId: string
	siteTouched: boolean
	categoryModifierId: string
	canPickModifier: boolean
	modifiers: PluginCategoryModifier[]
	preview: PluginModifierShort | null
	showPreview: boolean
	errors: FieldErrors
	onCategoryChange: (id: string) => void
	onSiteChange: (id: string) => void
	onModifierChange: (id: string) => void
	onApplyPreview: () => void
}

export function CategoryAndSiteSection({
	categories,
	sites,
	userSiteId,
	categoryId,
	siteId,
	siteTouched,
	categoryModifierId,
	canPickModifier,
	modifiers,
	preview,
	showPreview,
	errors,
	onCategoryChange,
	onSiteChange,
	onModifierChange,
	onApplyPreview,
}: CategoryAndSiteSectionProps) {
	const categorySections = useMemo(() => {
		const sections = new Map<string, {name: string | null; items: PluginCategory[]}>()
		categories.forEach(c => {
			if (c.categoryGroupId) {
				const key = `section:${c.categoryGroupId}`
				const name = c.categoryGroup?.name || ''
				const bucket = sections.get(key)
				if (bucket) {
					bucket.items.push(c)
				} else {
					sections.set(key, {name, items: [c]})
				}
				return
			}
			const key = `root:${c.id}`
			sections.set(key, {name: null, items: [c]})
		})
		return [...sections.values()]
	}, [categories])
	const selectedSite = sites.find(s => s.id === siteId) || null

	return (
		<section className='it-ticket-section'>
			<div className='it-ticket-section__head'>
				<div className='it-ticket-section__num'>1</div>
				<div>
					<div className='it-ticket-section__title'>Что случилось и где</div>
					<div className='it-ticket-section__subtitle'>Выберите категорию и площадку</div>
				</div>
			</div>
			<div className='it-ticket-section__body'>
				{categories.length === 0 ? <div className='it-f-label__hint'>Нет доступных категорий</div> : null}
				{sites.length === 0 ? <div className='it-f-label__hint'>Нет доступных площадок</div> : null}
				<div className='it-ticket-grid'>
					<label className='it-f-label'>
						<span className='it-f-label__text'>
							Категория <span className='it-f-required'>*</span>
						</span>
						<select
							className={`it-f${errors.categoryId ? ' it-f--error' : ''}`}
							value={categoryId}
							onChange={e => onCategoryChange(e.target.value)}
						>
							<option value=''>— Выберите категорию —</option>
							{categorySections.map(section =>
								section.name === null ? (
									// Категория без раздела — обычный пункт списка, вне групп.
									section.items.map(c => (
										<option key={c.id} value={c.id}>
											{c.name}
										</option>
									))
								) : (
									<optgroup key={section.name} label={section.name}>
										{section.items.map(c => (
											<option key={c.id} value={c.id}>
												{c.name}
											</option>
										))}
									</optgroup>
								),
							)}
						</select>
						{errors.categoryId ? <span className='it-f__err'>{errors.categoryId}</span> : null}
					</label>

					<label className='it-f-label'>
						<span className='it-f-label__text'>
							Площадка <span className='it-f-required'>*</span>
						</span>
						<select
							className={`it-f${errors.siteId ? ' it-f--error' : ''}`}
							value={siteId}
							onChange={e => onSiteChange(e.target.value)}
						>
							<option value=''>— Выберите площадку —</option>
							{sites.map((s: PluginSite) => (
								<option key={s.id} value={s.id}>
									{s.name}
								</option>
							))}
						</select>
						{errors.siteId ? <span className='it-f__err'>{errors.siteId}</span> : null}
						{selectedSite && selectedSite.address && (siteTouched || siteId !== userSiteId) ? (
							<span className='it-f-label__hint'>{selectedSite.address}</span>
						) : null}
					</label>

					{canPickModifier && modifiers.length > 0 ? (
						<div style={{gridColumn: '1 / -1'}}>
							<label className='it-f-label'>
								<span className='it-f-label__text'>Подкатегория</span>
								<select
									className='it-f'
									value={categoryModifierId}
									onChange={e => onModifierChange(e.target.value)}
								>
									<option value=''>— Без подкатегории —</option>
									{modifiers.map(m => (
										<option key={m.id} value={m.id}>
											{m.name}
										</option>
									))}
								</select>
								{showPreview && preview ? (
									<div className='it-modifier-preview-row'>
										<button type='button' className='it-modifier-preview' onClick={onApplyPreview}>
											{preview.name}
										</button>
										<span className='it-f-label__hint'>
											подберётся автоматически (нажмите, чтобы выбрать)
										</span>
									</div>
								) : (
									<span className='it-f-label__hint'>
										При выборе срок и приоритет подставятся из подкатегории
									</span>
								)}
							</label>
						</div>
					) : null}
				</div>
			</div>
		</section>
	)
}
