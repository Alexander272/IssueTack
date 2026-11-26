// Секция 2 формы создания: заголовок, описание и вложения. Файлы выбираются
// кнопкой или вставкой из буфера — добавление обрабатывает родитель (общая точка
// для обоих путей, с дедупликацией).

import React from 'react'

import {formatBytes} from '../../labels'
import {FileIcon} from '../icons'

interface DescriptionSectionProps {
	title: string
	description: string
	files: File[]
	seedWarning: string | null
	titleError?: string
	onTitleChange: (title: string) => void
	onDescriptionChange: (description: string) => void
	onAddFiles: (files: File[]) => void
	onRemoveFile: (file: File) => void
}

export function DescriptionSection({
	title,
	description,
	files,
	seedWarning,
	titleError,
	onTitleChange,
	onDescriptionChange,
	onAddFiles,
	onRemoveFile,
}: DescriptionSectionProps) {
	const onPickFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
		onAddFiles(Array.from(e.target.files || []))
		e.target.value = ''
	}

	return (
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
						className={`it-f${titleError ? ' it-f--error' : ''}`}
						type='text'
						value={title}
						maxLength={150}
						placeholder='Например: Не работает принтер в кабинете 305'
						onChange={e => onTitleChange(e.target.value)}
					/>
					{titleError ? <span className='it-f__err'>{titleError}</span> : null}
					{!titleError ? <span className='it-ticket-counter'>{title.length} / 150</span> : null}
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
						onChange={e => onDescriptionChange(e.target.value)}
					/>
				</label>

				<div className='it-f-label'>
					<span className='it-f-label__text'>Вложения</span>
					<input className='it-f it-f--file' type='file' multiple onChange={onPickFiles} />
					<div className='it-f-label__hint'>Можно вставить из буфера обмена (Ctrl+V)</div>
					{seedWarning ? (
						<div className='it-f-label__hint'>Вложения из сообщения: {seedWarning}.</div>
					) : null}
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
										onClick={() => onRemoveFile(f)}
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
	)
}
