import React, { useState } from 'react'

import { ApiError, createTicket } from '../api'
import type { PluginCategory, PluginContextResult, PluginCreateResult, PluginSite } from '../types'
import { formatBytes, priorityMeta } from '../labels'
import { FileIcon } from './icons'

interface CreateTabProps {
	channelId: string
	userId: string
	context: PluginContextResult
	onCreated: () => void
}

interface FieldErrors {
	title?: string
	categoryId?: string
	siteId?: string
}

export default function CreateTab({ channelId, userId, context, onCreated }: CreateTabProps) {
	const userSiteId =
		context.user?.siteId && (context.sites || []).some(s => s.id === context.user.siteId) ? context.user.siteId : ''
	const [title, setTitle] = useState('')
	const [description, setDescription] = useState('')
	const [categoryId, setCategoryId] = useState('')
	const [siteId, setSiteId] = useState(userSiteId)
	const [siteTouched, setSiteTouched] = useState(false)
	const [files, setFiles] = useState<File[]>([])
	const [submitting, setSubmitting] = useState(false)
	const [error, setError] = useState<string | null>(null)
	const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
	const [result, setResult] = useState<PluginCreateResult | null>(null)

	const categories = (context.categories || []).filter((c: PluginCategory) => c.isActive !== false)
	const sites = context.sites || []
	const selectedCategory = categories.find(c => c.id === categoryId) || null
	const selectedSite = sites.find(s => s.id === siteId) || null

	const reset = () => {
		setTitle('')
		setDescription('')
		setCategoryId('')
		setSiteId(userSiteId)
		setSiteTouched(false)
		setFiles([])
		setError(null)
		setFieldErrors({})
		setResult(null)
	}

	const onPickFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
		const picked = Array.from(e.target.files || [])
		setFiles(prev => {
			const seen = new Set(prev.map(f => `${f.name}:${f.size}`))
			return [...prev, ...picked.filter(f => !seen.has(`${f.name}:${f.size}`))]
		})
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
		setFieldErrors(errors)
		if (errors.title || errors.categoryId || errors.siteId) {
			return
		}
		setSubmitting(true)
		setError(null)
		try {
			const res = await createTicket(channelId, userId, {
				title: title.trim(),
				description: description.trim(),
				categoryId: categoryId || null,
				siteId: siteId || null,
				files,
			})
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
					{result.link ? (
						<a className='it-btn it-btn--primary' href={result.link} target='_blank' rel='noreferrer'>
							Перейти к заявке
						</a>
					) : null}
					<button type='button' className='it-btn' onClick={onCreated}>
						К моим заявкам
					</button>
					<button type='button' className='it-btn' onClick={reset}>
						Создать ещё
					</button>
				</div>
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
								{categories.map(c => (
									<option key={c.id} value={c.id}>
										{c.name}
									</option>
								))}
							</select>
							{fieldErrors.categoryId ? (
								<span className='it-f__err'>{fieldErrors.categoryId}</span>
							) : null}
							{selectedCategory ? (
								<span className='it-f-label__hint'>
									Приоритет: {priorityMeta(selectedCategory.priority).label}
								</span>
							) : null}
						</label>

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

			{error ? <div className='it-ticket-error'>{error}</div> : null}

			<div className='it-ticket-form__footer'>
				<button type='submit' className='it-btn it-btn--primary' disabled={submitting}>
					{submitting ? 'Создание…' : 'Создать заявку'}
				</button>
			</div>
		</form>
	)
}
