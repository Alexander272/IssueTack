import React, { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { PluginUserShort } from '../../types'

export interface NotifierFieldProps {
	options: PluginUserShort[]
	value: string[]
	onChange: (value: string[]) => void
	pluginUserLabel: (user: PluginUserShort) => string
	disabled?: boolean
}

interface PopperCoords {
	top: number
	left: number
	width: number
	maxHeight: number
	openUpward: boolean
}

export function NotifierField({ options, value, onChange, pluginUserLabel, disabled = false }: NotifierFieldProps) {
	const [isOpen, setIsOpen] = useState(false)
	const [search, setSearch] = useState('')
	const [displayValue, setDisplayValue] = useState('')
	const [extraCount, setExtraCount] = useState(0)
	const [coords, setCoords] = useState<PopperCoords | null>(null)

	const triggerRef = useRef<HTMLDivElement | null>(null)
	const popperRef = useRef<HTMLDivElement | null>(null)
	const triggerTextRef = useRef<HTMLSpanElement | null>(null)
	const searchInputRef = useRef<HTMLInputElement | null>(null)

	const mappedOptions = useMemo(() => {
		return options.map(u => ({ id: u.id, label: pluginUserLabel(u) }))
	}, [options, pluginUserLabel])

	const selectedItems = useMemo(() => {
		const map = new Map(mappedOptions.map(o => [o.id, o]))
		return value.map(id => map.get(id)).filter((item): item is { id: string; label: string } => Boolean(item))
	}, [mappedOptions, value])

	const filteredAndSortedOptions = useMemo(() => {
		const selectedSet = new Set(value)
		const sorted = [...selectedItems, ...mappedOptions.filter(o => !selectedSet.has(o.id))]
		if (!search.trim()) return sorted
		const query = search.toLowerCase()
		return sorted.filter(o => o.label.toLowerCase().includes(query))
	}, [mappedOptions, selectedItems, value, search])

	const fullValue = useMemo(() => selectedItems.map(v => v.label).join('; '), [selectedItems])

	const updatePosition = useCallback(() => {
		const trigger = triggerRef.current
		if (!trigger) return

		const rect = trigger.getBoundingClientRect()
		const padding = 8
		const preferredHeight = 365
		const spaceBelow = window.innerHeight - rect.bottom - padding
		const spaceAbove = rect.top - padding

		const openUpward = spaceBelow < preferredHeight && spaceAbove > spaceBelow
		const availableSpace = openUpward ? spaceAbove : spaceBelow
		const maxHeight = Math.max(160, Math.min(preferredHeight, availableSpace))

		const width = Math.max(rect.width, 240)
		const maxLeft = window.innerWidth - width - padding
		const left = Math.max(padding, Math.min(rect.left, maxLeft))

		setCoords({
			top: openUpward ? rect.top - 4 : rect.bottom + 4,
			left,
			width,
			maxHeight,
			openUpward,
		})
	}, [])

	useLayoutEffect(() => {
		if (!isOpen) return
		updatePosition()
		window.addEventListener('resize', updatePosition)
		window.addEventListener('scroll', updatePosition, true)
		return () => {
			window.removeEventListener('resize', updatePosition)
			window.removeEventListener('scroll', updatePosition, true)
		}
	}, [isOpen, updatePosition])

	useLayoutEffect(() => {
		const el = triggerTextRef.current
		if (!el) {
			setDisplayValue(fullValue)
			setExtraCount(0)
			return
		}

		const compute = () => {
			const target = triggerTextRef.current
			if (!target) return

			const labels = selectedItems.map(v => v.label)
			if (labels.length === 0) {
				setDisplayValue('')
				setExtraCount(0)
				return
			}

			const style = window.getComputedStyle(target)
			const canvas = document.createElement('canvas')
			const ctx = canvas.getContext('2d')
			if (!ctx) {
				setDisplayValue(fullValue)
				setExtraCount(0)
				return
			}

			ctx.font = `${style.fontWeight} ${style.fontSize} ${style.fontFamily}`
			const available = target.clientWidth
			const separator = '; '
			const sepWidth = ctx.measureText(separator).width
			const getBadgeReserve = (n: number) => ctx.measureText(`+${n}`).width + 24

			let used = 0
			let visible = 0

			for (let i = 0; i < labels.length; i++) {
				const text = labels[i] ?? ''
				const w = ctx.measureText(text).width + (i > 0 ? sepWidth : 0)
				const remaining = labels.length - i - 1
				const reserve = remaining > 0 ? getBadgeReserve(remaining) : 0

				if (used + w + reserve <= available) {
					used += w
					visible = i + 1
				} else {
					break
				}
			}

			if (visible === 0) {
				setDisplayValue(labels[0] ?? '')
				setExtraCount(labels.length - 1)
			} else {
				setDisplayValue(labels.slice(0, visible).join(separator))
				setExtraCount(labels.length - visible)
			}
		}

		compute()
		const ro = new ResizeObserver(compute)
		ro.observe(el)
		return () => ro.disconnect()
	}, [fullValue, selectedItems])

	useEffect(() => {
		if (!isOpen) return

		const handleClickOutside = (e: MouseEvent) => {
			const target = e.target as Node
			if (!triggerRef.current?.contains(target) && !popperRef.current?.contains(target)) {
				setIsOpen(false)
			}
		}

		const handleKeyDown = (e: KeyboardEvent) => {
			if (e.key === 'Escape') setIsOpen(false)
		}

		document.addEventListener('mousedown', handleClickOutside)
		document.addEventListener('keydown', handleKeyDown)
		return () => {
			document.removeEventListener('mousedown', handleClickOutside)
			document.removeEventListener('keydown', handleKeyDown)
		}
	}, [isOpen])

	useEffect(() => {
		if (isOpen) {
			setSearch('')
			requestAnimationFrame(() => searchInputRef.current?.focus())
		}
	}, [isOpen])

	const toggleOption = (id: string) => {
		onChange(value.includes(id) ? value.filter(v => v !== id) : [...value, id])
	}

	return (
		<div className='it-f-label'>
			<span className='it-f-label__text'>Уведомлять</span>
			{options.length > 0 ? (
				<div className='it-f-select-wrapper'>
					<div
						ref={triggerRef}
						className={`it-f-select-trigger ${isOpen ? 'is-open' : ''} ${disabled ? 'is-disabled' : ''}`}
						onClick={() => !disabled && setIsOpen(prev => !prev)}
						title={fullValue || undefined}
					>
						<span ref={triggerTextRef} className='it-f-select-trigger__text'>
							{displayValue || (
								<span className='it-f-select-trigger__placeholder'>Выберите пользователей...</span>
							)}
						</span>
						{extraCount > 0 && <span className='it-f-select-badge'>+{extraCount}</span>}
					</div>

					{isOpen &&
						coords &&
						createPortal(
							<div
								ref={popperRef}
								className='it-f-select-popper'
								style={{
									position: 'fixed',
									top: coords.top,
									left: coords.left,
									width: coords.width,
									maxHeight: coords.maxHeight,
									display: 'flex',
									flexDirection: 'column',
									transform: coords.openUpward ? 'translateY(-100%)' : 'none',
									zIndex: 9999,
								}}
							>
								<div className='it-f-select-search'>
									<svg
										className='it-f-select-search__icon'
										viewBox='0 0 24 24'
										width='16'
										height='16'
										stroke='currentColor'
										strokeWidth='2'
										fill='none'
									>
										<circle cx='11' cy='11' r='8' />
										<line x1='21' y1='21' x2='16.65' y2='16.65' />
									</svg>
									<input
										ref={searchInputRef}
										type='text'
										className='it-f-select-search__input'
										placeholder='Поиск'
										value={search}
										onChange={e => setSearch(e.target.value)}
									/>
								</div>

								<div
									className='it-f-select-listbox'
									style={{ flex: 1, minHeight: 0, overflowY: 'auto' }}
								>
									{filteredAndSortedOptions.length > 0 ? (
										filteredAndSortedOptions.map(option => {
											const isSelected = value.includes(option.id)
											return (
												<div
													key={option.id}
													className={`it-f-select-option ${isSelected ? 'is-selected' : ''}`}
													onClick={() => toggleOption(option.id)}
												>
													<div className='it-f-select-option__row'>
														<span
															className={`it-f-select-checkbox ${isSelected ? 'is-checked' : ''}`}
														>
															{isSelected && (
																<svg
																	viewBox='0 0 24 24'
																	width='14'
																	height='14'
																	stroke='currentColor'
																	strokeWidth='2.5'
																	fill='none'
																>
																	<polyline points='20 6 9 17 4 12' />
																</svg>
															)}
														</span>
														<span className='it-f-select-option__label'>
															{option.label}
														</span>
													</div>
													<div className='it-f-select-option__divider' />
												</div>
											)
										})
									) : (
										<div className='it-f-select-empty'>Ничего не найдено</div>
									)}
								</div>
							</div>,
							document.body,
						)}
				</div>
			) : (
				<span className='it-f-label__hint'>Нет доступных пользователей</span>
			)}
		</div>
	)
}
