import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { ApiError, createTicket } from '../api'
import type { CreateTicketPayload } from '../api'
import type {
	PluginCategory,
	PluginContextResult,
	PluginCreateResult,
	PluginScope,
	PluginSite,
	PluginUserShort,
	Priority,
} from '../types'
import { formatBytes } from '../labels'
import { clearDraft, loadDraft, saveDraft } from '../createDraft'
import { useClipboardFiles } from '../useClipboardFiles'
import { autoDueDateLocal, dueDateValueForSubmit } from '../workDays'
import { FileIcon } from './icons'
import { NotifierField } from './Form/NotifierField'

// pluginUserLabel — подпись пользователя в селектах формы: «Фамилия Имя
// (username)», без имён — просто username (как в диалогах бота).

interface CreateTabProps {
	scope: PluginScope
	context: PluginContextResult
	onCreated: () => void
	// onOpen открывает только что созданную заявку в карточке внутри этой же
	// модалки, без перехода в веб-приложение.
	onOpen: (ticketId: string) => void
	// onDraftChange сообщает родителю, есть ли непустой черновик: по нему
	// ModalController решает, можно ли закрывать окно кликом мимо/Escape.
	onDraftChange?: (hasDraft: boolean) => void
}

interface FieldErrors {
	title?: string
	categoryId?: string
	siteId?: string
	ownerId?: string
	priority?: string
}

// PRIORITY_ORDER — порядок карточек приоритета, PRIORITY_LABELS/COLORS/DESCRIPTIONS —
// как в вебе (PRIORITY_MAP + priorityDescriptions в TaskCreateForm).
const PRIORITY_ORDER: Priority[] = ['low', 'medium', 'high', 'urgent']
const PRIORITY_LABELS: Record<Priority, string> = {
	low: 'Низкий',
	medium: 'Средний',
	high: 'Высокий',
	urgent: 'Критичный',
}
const PRIORITY_COLORS: Record<Priority, string> = {
	low: '#10b981',
	medium: '#f59e0b',
	high: '#ef4444',
	urgent: '#dc2626',
}
const PRIORITY_DESCRIPTIONS: Record<Priority, string> = {
	low: 'Не мешает работе, можно подождать',
	medium: 'Работа затруднена, есть обходной путь',
	high: 'Работа полностью остановлена',
	urgent: 'Критично, требуется немедленная реакция',
}

function pluginUserLabel(u: PluginUserShort): string {
	if (u.lastName || u.firstName) return `${u.lastName} ${u.firstName}`.trim() + ` (${u.username})`
	return u.username
}



export default function CreateTab({ scope, context, onCreated, onOpen, onDraftChange }: CreateTabProps) {
	const userSiteId =
		context.user?.siteId && (context.sites || []).some(s => s.id === context.user.siteId) ? context.user.siteId : ''
	// Роли — зеркало веб-формы: isManager (начальник области ИЛИ управляет
	// группой) открывает «Расширенные настройки», исполнитель (не менеджер, но
	// участник групп) — секцию «Заказчик». Финальные права не здесь, а в
	// TicketService.Create, сервер применит ограничения даже при подмене полей.
	const isManager = Boolean(context.isManager)
	const isRealmAdmin = Boolean(context.isRealmAdmin)
	const isExecutor = !isManager && (context.memberGroupIds || []).length > 0
	const customers = context.customers || []
	const executors = context.executors || []
	const realmGroups = context.groups || []

	const [initialDraft] = useState(() => loadDraft(context.realmId, context.user.id))
	const [title, setTitle] = useState(initialDraft?.title ?? '')
	const [description, setDescription] = useState(initialDraft?.description ?? '')
	const [categoryId, setCategoryId] = useState(initialDraft?.categoryId ?? '')
	const [categoryModifierId, setCategoryModifierId] = useState(initialDraft?.categoryModifierId ?? '')
	const [siteId, setSiteId] = useState(initialDraft?.siteId || userSiteId)
	const [siteTouched, setSiteTouched] = useState(false)
	const [priority, setPriority] = useState(initialDraft?.priority ?? '')
	const [groupId, setGroupId] = useState(initialDraft?.groupId ?? '')
	const [assigneeId, setAssigneeId] = useState(initialDraft?.assigneeId ?? '')
	const [ownerId, setOwnerId] = useState(initialDraft?.ownerId ?? '')
	const [notifierIds, setNotifierIds] = useState<string[]>(initialDraft?.notifierIds ?? [])
	const [ownerMuted, setOwnerMuted] = useState(Boolean(initialDraft?.ownerMuted))
	const [dueDate, setDueDate] = useState(initialDraft?.dueDate ?? '')
	const [files, setFiles] = useState<File[]>([])
	const [submitting, setSubmitting] = useState(false)
	const [error, setError] = useState<string | null>(null)
	const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
	const [result, setResult] = useState<PluginCreateResult | null>(null)
	const [hasDraft, setHasDraft] = useState(false)
	const [confirmingClear, setConfirmingClear] = useState(false)

	// Автор и заказчик не выбираются наблюдателями (как excludeIds в веб-форме):
	// они и так получают уведомления. Если заказчика сменили на уже выбранного
	// наблюдателя, тот убирается из списка — иначе в запрос ушёл бы запрещённый id.
	const excludedNotifierIds = useMemo(() => {
		const ids = new Set<string>()
		if (context.user.id) ids.add(context.user.id)
		if (ownerId) ids.add(ownerId)
		return ids
	}, [context.user.id, ownerId])

	const notifierOptions = useMemo(
		() => customers.filter(u => !excludedNotifierIds.has(u.id)),
		[customers, excludedNotifierIds],
	)

	useEffect(() => {
		setNotifierIds(prev => {
			const next = prev.filter(id => !excludedNotifierIds.has(id))
			return next.length === prev.length ? prev : next
		})
	}, [excludedNotifierIds])

	useEffect(() => {
		if (!ownerId && ownerMuted) {
			setOwnerMuted(false)
		}
	}, [ownerId, ownerMuted])

	const categories = (context.categories || []).filter((c: PluginCategory) => c.isActive !== false)
	// Категории группируются по разделам (таксономия) — нативные optgroup.
	// Категория без раздела — отдельный блок из одного пункта (без группы
	// «Без раздела»), поэтому несколько таких категорий могут стоять в разных
	// местах между разделами.
	const categorySections = useMemo(() => {
		const sections = new Map<string, { name: string | null; items: PluginCategory[] }>()
		categories.forEach(c => {
			if (c.categoryGroupId) {
				const key = `section:${c.categoryGroupId}`
				const name = c.categoryGroup?.name || ''
				const bucket = sections.get(key)
				if (bucket) {
					bucket.items.push(c)
				} else {
					sections.set(key, { name, items: [c] })
				}
				return
			}
			const key = `root:${c.id}`
			sections.set(key, { name: null, items: [c] })
		})
		return [...sections.values()]
	}, [categories])
	const sites = context.sites || []
	const selectedSite = sites.find(s => s.id === siteId) || null
	// Подкатегорию выбирает исполнитель (участник любой группы реалма) или
	// начальник области. Право не привязано к assignee новой заявки: исполнитель
	// видит поле, даже если в форме выбран другой исполнитель.
	const canPickModifier = isRealmAdmin || (context.memberGroupIds || []).length > 0
	const modifiers = (categories.find(c => c.id === categoryId)?.modifiers || []).filter(m => m.isActive !== false)

	// Авто-назначение в стиле веба (AdvancedSettingsSection + onChange категории
	// в TaskCreateForm): смена категории подставляет группу и дефолтный приоритет
	// категории, смена группы — дефолтного исполнителя. Срабатывают только на
	// реальном изменении поля (useRef-пред), а не на первом рендере или
	// восстановлении черновика — иначе авто-подстановка затирала бы сохранённые
	// значения. Сервер всё равно авторитетен: подставленные здесь значения
	// совпадают с теми, что TicketService.Create применил бы сам при пустых полях.
	const prevCategoryId = useRef(categoryId)
	const prevGroupId = useRef(groupId)
	const prevModifierCategoryId = useRef(categoryId)
	const prevCategoryModifierId = useRef(categoryModifierId)

	// Подкатегория принадлежит категории: при смене категории форма сбрасывает её
	// следом за сервером (иначе в запрос ушёл бы чужой id).
	useEffect(() => {
		if (prevModifierCategoryId.current === categoryId) return
		prevModifierCategoryId.current = categoryId
		setCategoryModifierId('')
	}, [categoryId])

	// Эффект 1: категория → группа + приоритет + срок (менеджер И исполнитель).
	// Исполнитель создаёт заявку и должен получать те же авто-подстановки, что и менеджер.
	useEffect(() => {
		if (!(isManager || isExecutor)) return
		if (prevCategoryId.current === categoryId) return
		prevCategoryId.current = categoryId
		const cat = categories.find(c => c.id === categoryId)
		setGroupId(cat?.groupId ?? '')
		if (cat?.priority) setPriority(cat.priority)
		// Автосрок из категории — зеркало веба/бэкенда: дефолт категории
		// проставляется в поле срока в рабочих днях (с учётом выходных и праздников,
		// конец целевого рабочего дня в календарной TZ), а категория без автосрока
		// его сбрасывает, иначе в форме остался бы срок предыдущей категории.
		setDueDate(autoDueDateLocal(cat?.defaultDueDays, context.calendar) ?? '')
	}, [categoryId, categories, isManager, isExecutor, context.calendar])

	// Эффект 1а: подкатегория → приоритет + срок. Как и в веб-форме, значения
	// берутся из подкатегории, а пустые поля наследуются от категории. Срабатывает
	// только на реальном выборе пользователя, не на восстановлении черновика.
	useEffect(() => {
		if (prevCategoryModifierId.current === categoryModifierId) return
		prevCategoryModifierId.current = categoryModifierId
		if (!categoryModifierId) return
		const modifier = (categories.find(c => c.id === categoryId)?.modifiers || []).find(
			m => m.id === categoryModifierId,
		)
		if (!modifier) return
		const cat = categories.find(c => c.id === categoryId)
		setPriority(modifier.priority || cat?.priority || '')
		const days = modifier.defaultDueDays ?? cat?.defaultDueDays
		setDueDate(days != null && days > 0 ? autoDueDateLocal(days, context.calendar) ?? '' : '')
	}, [categoryModifierId, categories, categoryId, context.calendar])

	useEffect(() => {
		if (!isManager) return
		if (prevGroupId.current === groupId) return
		prevGroupId.current = groupId
		if (!groupId) {
			setAssigneeId('')
			return
		}
		const group = realmGroups.find(g => g.id === groupId)
		setAssigneeId(group?.defaultAssigneeId ?? '')
	}, [groupId, realmGroups, isManager])

	useEffect(() => {
		saveDraft(context.realmId, context.user.id, {
			title,
			description,
			categoryId,
			categoryModifierId,
			siteId,
			priority,
			groupId,
			assigneeId,
			ownerId,
			notifierIds,
			ownerMuted,
			dueDate,
		})
	}, [
		context.realmId,
		context.user.id,
		title,
		description,
		categoryId,
		categoryModifierId,
		siteId,
		priority,
		groupId,
		assigneeId,
		ownerId,
		notifierIds,
		ownerMuted,
		dueDate,
	])

	useEffect(() => {
		// Дефолтная площадка пользователя сама по себе не считается черновиком.
		const hasContent = Boolean(
			title.trim() ||
			description.trim() ||
			categoryId ||
			categoryModifierId ||
			files.length > 0 ||
			(siteId && siteId !== userSiteId) ||
			priority ||
			groupId ||
			assigneeId ||
			ownerId ||
			dueDate ||
			notifierIds.length > 0 ||
			ownerMuted,
		)
		setHasDraft(hasContent)
		onDraftChange?.(hasContent)
	}, [
		title,
		description,
		categoryId,
		categoryModifierId,
		siteId,
		files,
		userSiteId,
		onDraftChange,
		priority,
		groupId,
		assigneeId,
		ownerId,
		notifierIds,
		ownerMuted,
		dueDate,
	])

	const reset = () => {
		clearDraft(context.realmId, context.user.id)
		setTitle('')
		setDescription('')
		setCategoryId('')
		setCategoryModifierId('')
		setSiteId(userSiteId)
		setSiteTouched(false)
		setPriority('')
		setGroupId('')
		setAssigneeId('')
		setOwnerId('')
		setOwnerMuted(false)
		setDueDate('')
		setNotifierIds([])
		setFiles([])
		setError(null)
		setFieldErrors({})
		setResult(null)
	}

	// addFiles — общая точка для выбора файлов и вставки из буфера: вложения
	// дедуплицируются по имени и размеру (как в вебе).
	const addFiles = useCallback((picked: File[]) => {
		if (picked.length === 0) return
		setFiles(prev => {
			const seen = new Set(prev.map(f => `${f.name}:${f.size}`))
			return [...prev, ...picked.filter(f => !seen.has(`${f.name}:${f.size}`))]
		})
	}, [])

	useClipboardFiles(addFiles)

	const onPickFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
		addFiles(Array.from(e.target.files || []))
		e.target.value = ''
	}

	const clearFieldError = (field: keyof FieldErrors) => {
		setFieldErrors(prev => ({ ...prev, [field]: undefined }))
	}

	const submit = async (e: React.FormEvent) => {
		e.preventDefault()
		const errors: FieldErrors = {}
		if (!title.trim()) {
			errors.title = 'Обязательное поле'
		}
		if (!categoryId) {
			errors.categoryId = 'Обязательное поле'
		}
		if (!siteId) {
			errors.siteId = 'Обязательное поле'
		}
		if ((isManager || isExecutor) && !priority) {
			errors.priority = 'Обязательное поле'
		}
		setFieldErrors(errors)
		if (errors.title || errors.categoryId || errors.siteId || errors.priority) {
			return
		}
		setSubmitting(true)
		setError(null)
		try {
			// Поля доп. секций отправляются только тем, кому они доступны:
			// менеджер — все (приоритет, группа, исполнитель, заявитель, срок),
			// исполнитель — заказчик и самоназначение. Обычный заявитель может
			// указать наблюдателей. Группа/приоритет берутся с сервера из категории.
			const payload: CreateTicketPayload = {
				title: title.trim(),
				description: description.trim(),
				categoryId: categoryId || null,
				...(canPickModifier && categoryModifierId ? {categoryModifierId} : {}),
				siteId: siteId || null,
				files,
			}
			if (isManager) {
				payload.priority = priority
				if (groupId) payload.groupId = groupId
				if (assigneeId) payload.assigneeId = assigneeId
				if (ownerId) payload.ownerId = ownerId
				if (dueDate) payload.dueDate = dueDateValueForSubmit(dueDate, context.calendar)
				if (ownerMuted && ownerId) payload.ownerMuted = true
				payload.notifierIds = notifierIds
			} else if (isExecutor) {
				if (ownerId) payload.ownerId = ownerId
				if (assigneeId) payload.assigneeId = assigneeId
				if (priority) payload.priority = priority
				if (dueDate) payload.dueDate = dueDateValueForSubmit(dueDate, context.calendar)
				payload.notifierIds = notifierIds
			} else if (notifierIds.length > 0) {
				payload.notifierIds = notifierIds
			}
			const res = await createTicket(scope, payload)
			// Заявка создана — черновик больше не нужен, а поля очищаем, чтобы
			// «Создать ещё» и закрытие окна не тянули старые данные.
			clearDraft(context.realmId, context.user.id)
			setTitle('')
			setDescription('')
			setCategoryId('')
			setCategoryModifierId('')
			setSiteId(userSiteId)
			setSiteTouched(false)
			setPriority('')
			setGroupId('')
			setAssigneeId('')
			setOwnerId('')
			setOwnerMuted(false)
			setNotifierIds([])
			setDueDate('')
			setFiles([])
			setResult({ number: res.number, link: res.link, ...res })
		} catch (err) {
			setError(err instanceof ApiError ? err.message : 'Не удалось создать заявку')
		} finally {
			setSubmitting(false)
		}
	}

	if (result) {
		return (
			<div className='it-ticket-success'>
				<div className='it-ticket-success__title'>Заявка №{result.number || ''} создана</div>
				<div className='it-ticket-success__actions'>
					<button type='button' className='it-btn it-btn--primary' onClick={() => onOpen(result.id)}>
						Открыть заявку
					</button>
					<button type='button' className='it-btn' onClick={onCreated}>
						К моим заявкам
					</button>
					<button type='button' className='it-btn' onClick={reset}>
						Создать ещё
					</button>
				</div>
				{result.link ? (
					<a className='it-ticket-success__link' href={result.link} target='_blank' rel='noreferrer'>
						Открыть в веб-приложении
					</a>
				) : null}
			</div>
		)
	}

	return (
		<form className='it-ticket-form' onSubmit={submit}>
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
								className={`it-f${fieldErrors.categoryId ? ' it-f--error' : ''}`}
								value={categoryId}
								onChange={e => {
									setCategoryId(e.target.value)
									clearFieldError('categoryId')
								}}
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
							{fieldErrors.categoryId ? (
								<span className='it-f__err'>{fieldErrors.categoryId}</span>
							) : null}
						</label>

						{canPickModifier && modifiers.length > 0 ? (
							<label className='it-f-label'>
								<span className='it-f-label__text'>Подкатегория</span>
								<select
									className='it-f'
									value={categoryModifierId}
									onChange={e => setCategoryModifierId(e.target.value)}
								>
									<option value=''>— Без подкатегории —</option>
									{modifiers.map(m => (
										<option key={m.id} value={m.id}>
											{m.name}
										</option>
									))}
								</select>
								<span className='it-f-label__hint'>
									При выборе срок и приоритет подставятся из подкатегории
								</span>
							</label>
						) : null}

						<label className='it-f-label'>
							<span className='it-f-label__text'>
								Площадка <span className='it-f-required'>*</span>
							</span>
							<select
								className={`it-f${fieldErrors.siteId ? ' it-f--error' : ''}`}
								value={siteId}
								onChange={e => {
									setSiteId(e.target.value)
									setSiteTouched(true)
									clearFieldError('siteId')
								}}
							>
								<option value=''>— Выберите площадку —</option>
								{sites.map((s: PluginSite) => (
									<option key={s.id} value={s.id}>
										{s.name}
									</option>
								))}
							</select>
							{fieldErrors.siteId ? <span className='it-f__err'>{fieldErrors.siteId}</span> : null}
							{selectedSite && selectedSite.address && (siteTouched || siteId !== userSiteId) ? (
								<span className='it-f-label__hint'>{selectedSite.address}</span>
							) : null}
						</label>
					</div>
				</div>
			</section>

			<section className='it-ticket-section'>
				<div className='it-ticket-section__head'>
					<div className='it-ticket-section__num'>2</div>
					<div>
						<div className='it-ticket-section__title'>Описание проблемы</div>
						<div className='it-ticket-section__subtitle'>
							Чем подробнее вы опишете проблему, тем быстрее её решат
						</div>
					</div>
				</div>
				<div className='it-ticket-section__body'>
					<label className='it-f-label'>
						<span className='it-f-label__text'>
							Заголовок <span className='it-f-required'>*</span>
						</span>
						<input
							className={`it-f${fieldErrors.title ? ' it-f--error' : ''}`}
							type='text'
							value={title}
							maxLength={150}
							placeholder='Например: Не работает принтер в кабинете 305'
							onChange={e => {
								setTitle(e.target.value)
								clearFieldError('title')
							}}
						/>
						{fieldErrors.title ? <span className='it-f__err'>{fieldErrors.title}</span> : null}
						{!fieldErrors.title ? <span className='it-ticket-counter'>{title.length} / 150</span> : null}
					</label>

					<label className='it-f-label'>
						<span className='it-f-label__text'>Подробное описание</span>
						<textarea
							className='it-f it-f--area'
							rows={6}
							value={description}
							maxLength={5000}
							placeholder={
								'Опишите, что произошло:\n• Что вы делали перед проблемой?\n• Что именно не работает?\n• Появляются ли ошибки?\n• Когда это началось?'
							}
							onChange={e => setDescription(e.target.value)}
						/>
					</label>

					<div className='it-f-label'>
						<span className='it-f-label__text'>Вложения</span>
						<input className='it-f it-f--file' type='file' multiple onChange={onPickFiles} />
						<div className='it-f-label__hint'>Можно вставить из буфера обмена (Ctrl+V)</div>
						{files.length > 0 ? (
							<ul className='it-ticket-files'>
								{files.map(f => (
									<li key={`${f.name}:${f.size}`} className='it-ticket-file'>
										<FileIcon size={14} />
										<span className='it-ticket-file__name'>{f.name}</span>
										<span className='it-ticket-file__size'>{formatBytes(f.size)}</span>
										<button
											type='button'
											className='it-ticket-file__remove'
											aria-label={`Удалить ${f.name}`}
											onClick={() => setFiles(prev => prev.filter(x => x !== f))}
										>
											×
										</button>
									</li>
								))}
							</ul>
						) : null}
					</div>
				</div>
			</section>

			{isManager || isExecutor ? (
				<section className='it-ticket-section'>
					<div className='it-ticket-section__head'>
						<div className='it-ticket-section__num'>3</div>
						<div>
							<div className='it-ticket-section__title'>
								{isManager ? 'Расширенные настройки' : 'Заказчик и сроки'}
							</div>
							<div className='it-ticket-section__subtitle'>
								{isManager
									? 'Доступно менеджерам'
									: 'Укажите заказчика и желаемый срок выполнения. При необходимости назначьте себя исполнителем'}
							</div>
						</div>
					</div>
					<div className='it-ticket-section__body'>
						<div className='it-f-label'>
							<span className='it-f-label__text'>
								Приоритет <span className='it-f-required'>*</span>
							</span>
							<div className='it-f-priority' role='radiogroup' aria-label='Приоритет'>
								{PRIORITY_ORDER.map(value => (
									<button
										key={value}
										type='button'
										role='radio'
										aria-checked={priority === value}
										className={`it-f-priority-card${priority === value ? ' it-f-priority-card--active' : ''}`}
										onClick={() => {
											setPriority(value)
											clearFieldError('priority')
										}}
									>
										<span className='it-f-priority-card__head'>
											<span
												className='it-f-priority-card__dot'
												style={{ background: PRIORITY_COLORS[value] }}
											/>
											<span className='it-f-priority-card__label'>{PRIORITY_LABELS[value]}</span>
										</span>
										<span className='it-f-priority-card__desc'>{PRIORITY_DESCRIPTIONS[value]}</span>
									</button>
								))}
							</div>
							{fieldErrors.priority ? <span className='it-f__err'>{fieldErrors.priority}</span> : null}
						</div>

						<div className='it-ticket-grid'>
							<div className='it-f-label'>
								<label className='it-f-label' style={{ gap: 4 }}>
									<span className='it-f-label__text'>Заказчик</span>
									<select
										className={`it-f${fieldErrors.ownerId ? ' it-f--error' : ''}`}
										value={ownerId}
										onChange={e => {
											setOwnerId(e.target.value)
											clearFieldError('ownerId')
										}}
									>
										<option value=''>
											{isManager ? '— Заявитель: создатель —' : '— Не указан —'}
										</option>
										{customers.map(u => (
											<option key={u.id} value={u.id}>
												{pluginUserLabel(u)}
											</option>
										))}
									</select>
									{fieldErrors.ownerId ? (
										<span className='it-f__err'>{fieldErrors.ownerId}</span>
									) : null}
								</label>
							</div>

							<NotifierField
								options={notifierOptions}
								value={notifierIds}
								onChange={setNotifierIds}
								pluginUserLabel={pluginUserLabel}
							/>

							{isManager && ownerId ? (
								<div style={{ gridColumn: '1 / -1', marginTop: '-16px' }}>
									<label className='it-f-check' style={{ display: 'flex' }}>
										<input
											className='it-f-check__box'
											type='checkbox'
											checked={ownerMuted}
											onChange={e => setOwnerMuted(e.target.checked)}
										/>
										<span className='it-f-check__text'>
											Не уведомлять заказчика и скрыть заявку из его списка
										</span>
									</label>
								</div>
							) : null}

							<label className='it-f-label'>
								<span className='it-f-label__text'>Группа</span>
								<select
									className='it-f'
									value={groupId}
									onChange={e => setGroupId(e.target.value)}
									disabled={!isManager}
								>
									<option value=''>— Авто (по категории) —</option>
									{realmGroups.map(g => (
										<option key={g.id} value={g.id}>
											{g.name}
										</option>
									))}
								</select>
							</label>

							<label className='it-f-label'>
								<span className='it-f-label__text'>Исполнитель</span>
								<select
									className='it-f'
									value={assigneeId}
									onChange={e => setAssigneeId(e.target.value)}
								>
									<option value=''>— Авто (по группе) —</option>
									{executors.map(u => (
										<option key={u.id} value={u.id}>
											{pluginUserLabel(u)}
										</option>
									))}
								</select>
								{executors.length === 0 ? (
									<span className='it-f-label__hint'>Нет исполнителей</span>
								) : null}
								{isManager ? null : (
									<label className='it-f-label it-f-check'>
										<input
											className='it-f-check__box'
											type='checkbox'
											checked={assigneeId === context.user.id}
											onChange={e => setAssigneeId(e.target.checked ? context.user.id : '')}
										/>
										<span className='it-f-check__text'>Назначить меня исполнителем</span>
									</label>
								)}
							</label>

							<div style={{ gridColumn: '1 / -1' }}>
								<label className='it-f-label'>
									<span className='it-f-label__text'>Срок выполнения</span>
									<input
										className='it-f'
										type='datetime-local'
										value={dueDate}
										onChange={e => setDueDate(e.target.value)}
									/>
								</label>
							</div>
						</div>
					</div>
				</section>
			) : null}

			{!isManager && !isExecutor ? (
				<section className='it-ticket-section'>
					<div className='it-ticket-section__head'>
						<div className='it-ticket-section__num'>3</div>
						<div>
							<div className='it-ticket-section__title'>Уведомлять дополнительно</div>
							<div className='it-ticket-section__subtitle'>
								Кому еще сообщать о создании, смене статуса и комментариях. Ответственные по заявке
								получат эти уведомления автоматически.
							</div>
						</div>
					</div>
					<div className='it-ticket-section__body'>
						<NotifierField
							options={notifierOptions}
							value={notifierIds}
							onChange={setNotifierIds}
							pluginUserLabel={pluginUserLabel}
							label={null}
						/>
					</div>
				</section>
			) : null}

			{error ? <div className='it-ticket-error'>{error}</div> : null}

			<div className='it-ticket-form__footer'>
				{hasDraft ? (
					<button
						type='button'
						className='it-btn it-btn--danger'
						disabled={submitting}
						onClick={() => setConfirmingClear(true)}
					>
						Очистить
					</button>
				) : null}
				<button type='submit' className='it-btn it-btn--primary' disabled={submitting}>
					{submitting ? 'Создание…' : 'Создать заявку'}
				</button>
			</div>

			{confirmingClear ? (
				<div className='it-confirm' onClick={() => setConfirmingClear(false)}>
					<div className='it-confirm__box' onClick={e => e.stopPropagation()}>
						<div className='it-confirm__title'>Очистить черновик?</div>
						<div className='it-confirm__text'>Заполненные данные и вложения будут потеряны.</div>
						<div className='it-confirm__actions'>
							<button
								type='button'
								className='it-btn it-btn--sm'
								onClick={() => setConfirmingClear(false)}
							>
								Нет
							</button>
							<button
								type='button'
								className='it-btn it-btn--sm it-btn--danger it-btn--solid'
								onClick={() => {
									setConfirmingClear(false)
									reset()
								}}
							>
								Очистить
							</button>
						</div>
					</div>
				</div>
			) : null}
		</form>
	)
}
