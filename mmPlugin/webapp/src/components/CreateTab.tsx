// Форма создания заявки в плагине. Компонент — координатор: держит состояние,
// черновик и отправку; поля разложены по секциям, а побочные эффекты (вложения
// из сообщения, подбор подкатегории, авто-назначение) — по хукам в ./CreateTab.

import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react'

import {ApiError, createTicket} from '../api'
import type {CreateTicketSeed, PluginContextResult, PluginCreateResult, PluginScope} from '../types'
import {clearDraft, loadDraft, saveDraft} from '../createDraft'
import {useClipboardFiles} from '../useClipboardFiles'
import {AdvancedSection} from './CreateTab/AdvancedSection'
import {buildCreatePayload} from './CreateTab/buildPayload'
import {CategoryAndSiteSection} from './CreateTab/CategoryAndSiteSection'
import {DescriptionSection} from './CreateTab/DescriptionSection'
import {ClearConfirmDialog, SuccessView} from './CreateTab/Feedback'
import {NotifierSection} from './CreateTab/NotifierSection'
import {type AdvancedValues, type FieldErrors} from './CreateTab/model'
import {useAutoAssign} from './CreateTab/useAutoAssign'
import {useModifierPreview} from './CreateTab/useModifierPreview'
import {useSeedFiles} from './CreateTab/useSeedFiles'

interface CreateTabProps {
	scope: PluginScope
	context: PluginContextResult
	// Предзаполнение из сообщения. Перебивает только свои поля: описание,
	// заказчика (если пользователь вправе его выбрать) и вложения. Остальное
	// берём из сохранённого черновика.
	initialSeed?: CreateTicketSeed | null
	onCreated: () => void
	// onOpen открывает только что созданную заявку в карточке внутри этой же
	// модалки, без перехода в веб-приложение.
	onOpen: (ticketId: string) => void
	// onDraftChange сообщает родителю, есть ли непустой черновик: по нему
	// ModalController решает, можно ли закрывать окно кликом мимо/Escape.
	onDraftChange?: (hasDraft: boolean) => void
}

export default function CreateTab({scope, context, initialSeed, onCreated, onOpen, onDraftChange}: CreateTabProps) {
	const userSiteId =
		context.user?.siteId && (context.sites || []).some(s => s.id === context.user.siteId) ? context.user.siteId : ''
	// Роли — зеркало веб-формы: isManager (начальник области ИЛИ управляет
	// группой) открывает «Расширенные настройки», исполнитель (не менеджер, но
	// участник групп) — секцию «Заказчик». Финальные права не здесь, а в
	// TicketService.Create, сервер применит ограничения даже при подмене полей.
	const isManager = Boolean(context.isManager)
	const isRealmAdmin = Boolean(context.isRealmAdmin)
	const isExecutor = !isManager && (context.memberGroupIds || []).length > 0
	const customers = useMemo(() => context.customers || [], [context.customers])
	const executors = useMemo(() => context.executors || [], [context.executors])
	const realmGroups = useMemo(() => context.groups || [], [context.groups])
	const categories = useMemo(
		() => (context.categories || []).filter(c => c.isActive !== false),
		[context.categories],
	)

	// Автор сообщения становится заказчиком только у тех, кто вправе выбирать
	// заказчика (менеджер/исполнитель), и только если он найден в реалме по
	// mattermostId. Иначе остаётся значение из черновика.
	const seedOwnerId = useMemo(() => {
		if (!(isManager || isExecutor) || !initialSeed?.authorMmId) return ''
		const author = customers.find(u => u.mattermostId && u.mattermostId === initialSeed.authorMmId)
		return author?.id || ''
	}, [isManager, isExecutor, initialSeed, customers])
	const seedOwnerApplied = useRef(Boolean(seedOwnerId))

	const [initialDraft] = useState(() => loadDraft(context.realmId, context.user.id))
	const [title, setTitle] = useState(initialDraft?.title ?? '')
	const [description, setDescription] = useState(initialSeed?.description ?? initialDraft?.description ?? '')
	const [categoryId, setCategoryId] = useState(initialDraft?.categoryId ?? '')
	const [categoryModifierId, setCategoryModifierId] = useState(initialDraft?.categoryModifierId ?? '')
	const [siteId, setSiteId] = useState(initialDraft?.siteId || userSiteId)
	const [siteTouched, setSiteTouched] = useState(false)
	const [priority, setPriority] = useState(initialDraft?.priority ?? '')
	const [groupId, setGroupId] = useState(initialDraft?.groupId ?? '')
	const [assigneeId, setAssigneeId] = useState(initialDraft?.assigneeId ?? '')
	const [ownerId, setOwnerId] = useState(seedOwnerId || initialDraft?.ownerId || '')
	const [notifierIds, setNotifierIds] = useState<string[]>(initialDraft?.notifierIds ?? [])
	const [ownerMuted, setOwnerMuted] = useState(Boolean(initialDraft?.ownerMuted))
	const [dueDate, setDueDate] = useState(initialDraft?.dueDate ?? '')
	const {files, setFiles, seedWarning} = useSeedFiles(initialSeed)
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

	// Контекст модалки может подтянуться свежим уже после открытия формы —
	// подставляем заказчика один раз, когда список реалма с mattermostId готов.
	useEffect(() => {
		if (seedOwnerApplied.current || !seedOwnerId) return
		seedOwnerApplied.current = true
		setOwnerId(seedOwnerId)
	}, [seedOwnerId])

	const sites = context.sites || []
	// Подкатегорию выбирает исполнитель (участник любой группы реалма) или
	// начальник области. Право не привязано к assignee новой заявки: исполнитель
	// видит поле, даже если в форме выбран другой исполнитель.
	const canPickModifier = isRealmAdmin || (context.memberGroupIds || []).length > 0
	const modifiers = (categories.find(c => c.id === categoryId)?.modifiers || []).filter(m => m.isActive !== false)

	const {showPreview, preview, reset: resetPreview, applyPreview} = useModifierPreview({
		scope,
		categoryId,
		title,
		description,
		enabled: canPickModifier && !!categoryId && modifiers.length > 0,
		selectedId: categoryModifierId,
		onSelect: setCategoryModifierId,
	})

	useAutoAssign({
		categoryId,
		categoryModifierId,
		groupId,
		categories,
		groups: realmGroups,
		calendar: context.calendar,
		isManager,
		isExecutor,
		onModifierReset: resetPreview,
		setCategoryModifierId,
		setGroupId,
		setPriority,
		setDueDate,
		setAssigneeId,
	})

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
		resetPreview()
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
	}, [setFiles])

	useClipboardFiles(addFiles)

	const clearFieldError = useCallback((field: keyof FieldErrors) => {
		setFieldErrors(prev => ({...prev, [field]: undefined}))
	}, [])

	const patchAdvanced = useCallback((patch: Partial<AdvancedValues>) => {
		if (patch.priority !== undefined) setPriority(patch.priority)
		if (patch.ownerId !== undefined) setOwnerId(patch.ownerId)
		if (patch.ownerMuted !== undefined) setOwnerMuted(patch.ownerMuted)
		if (patch.groupId !== undefined) setGroupId(patch.groupId)
		if (patch.assigneeId !== undefined) setAssigneeId(patch.assigneeId)
		if (patch.dueDate !== undefined) setDueDate(patch.dueDate)
		if (patch.notifierIds !== undefined) setNotifierIds(patch.notifierIds)
	}, [])

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
			const payload = buildCreatePayload({
				isManager,
				isExecutor,
				canPickModifier,
				title,
				description,
				categoryId,
				categoryModifierId,
				siteId,
				files,
				calendar: context.calendar,
				advanced: {priority, ownerId, ownerMuted, groupId, assigneeId, dueDate, notifierIds},
			})
			const res = await createTicket(scope, payload)
			// Заявка создана — черновик больше не нужен, а поля очищаем, чтобы
			// «Создать ещё» и закрытие окна не тянули старые данные.
			reset()
			setResult({number: res.number, link: res.link, ...res})
		} catch (err) {
			setError(err instanceof ApiError ? err.message : 'Не удалось создать заявку')
		} finally {
			setSubmitting(false)
		}
	}

	if (result) {
		return <SuccessView result={result} onOpen={onOpen} onCreated={onCreated} onReset={reset} />
	}

	return (
		<form className='it-ticket-form' onSubmit={submit}>
			<CategoryAndSiteSection
				categories={categories}
				sites={sites}
				userSiteId={userSiteId}
				categoryId={categoryId}
				siteId={siteId}
				siteTouched={siteTouched}
				categoryModifierId={categoryModifierId}
				canPickModifier={canPickModifier}
				modifiers={modifiers}
				preview={preview}
				showPreview={showPreview}
				errors={fieldErrors}
				onCategoryChange={id => {
					setCategoryId(id)
					clearFieldError('categoryId')
				}}
				onSiteChange={id => {
					setSiteId(id)
					setSiteTouched(true)
					clearFieldError('siteId')
				}}
				onModifierChange={setCategoryModifierId}
				onApplyPreview={applyPreview}
			/>

			<DescriptionSection
				title={title}
				description={description}
				files={files}
				seedWarning={seedWarning}
				titleError={fieldErrors.title}
				onTitleChange={value => {
					setTitle(value)
					clearFieldError('title')
				}}
				onDescriptionChange={setDescription}
				onAddFiles={addFiles}
				onRemoveFile={file => setFiles(prev => prev.filter(x => x !== file))}
			/>

			{isManager || isExecutor ? (
				<AdvancedSection
					isManager={isManager}
					currentUserId={context.user.id}
					customers={customers}
					executors={executors}
					groups={realmGroups}
					notifierOptions={notifierOptions}
					values={{priority, ownerId, ownerMuted, groupId, assigneeId, dueDate, notifierIds}}
					errors={fieldErrors}
					onChange={patchAdvanced}
					onClearError={clearFieldError}
				/>
			) : null}

			{!isManager && !isExecutor ? (
				<NotifierSection options={notifierOptions} value={notifierIds} onChange={setNotifierIds} />
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
				<ClearConfirmDialog
					onCancel={() => setConfirmingClear(false)}
					onConfirm={() => {
						setConfirmingClear(false)
						reset()
					}}
				/>
			) : null}
		</form>
	)
}
