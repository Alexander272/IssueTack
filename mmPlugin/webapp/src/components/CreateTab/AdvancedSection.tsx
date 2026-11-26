// Секция 3 для менеджера и исполнителя: приоритет, заказчик, наблюдатели,
// группа, исполнитель и срок. Права показа зеркалят веб-форму; финальные права
// всё равно у TicketService.Create. Поля меняются через patch, чтобы родителю
// не приходилось держать дюжину отдельных сеттеров для дочерних компонентов.

import React from 'react'

import type {PluginGroupShort, PluginUserShort} from '../../types'
import {NotifierField} from '../Form/NotifierField'
import {
	pluginUserLabel,
	PRIORITY_COLORS,
	PRIORITY_DESCRIPTIONS,
	PRIORITY_LABELS,
	PRIORITY_ORDER,
	type AdvancedValues,
	type FieldErrors,
} from './model'

interface AdvancedSectionProps {
	isManager: boolean
	currentUserId: string
	customers: PluginUserShort[]
	executors: PluginUserShort[]
	groups: PluginGroupShort[]
	notifierOptions: PluginUserShort[]
	values: AdvancedValues
	errors: FieldErrors
	onChange: (patch: Partial<AdvancedValues>) => void
	onClearError: (field: keyof FieldErrors) => void
}

export function AdvancedSection({
	isManager,
	currentUserId,
	customers,
	executors,
	groups,
	notifierOptions,
	values,
	errors,
	onChange,
	onClearError,
}: AdvancedSectionProps) {
	const {priority, ownerId, ownerMuted, groupId, assigneeId, dueDate, notifierIds} = values

	return (
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
									onChange({priority: value})
									onClearError('priority')
								}}
							>
								<span className='it-f-priority-card__head'>
									<span
										className='it-f-priority-card__dot'
										style={{background: PRIORITY_COLORS[value]}}
									/>
									<span className='it-f-priority-card__label'>{PRIORITY_LABELS[value]}</span>
								</span>
								<span className='it-f-priority-card__desc'>{PRIORITY_DESCRIPTIONS[value]}</span>
							</button>
						))}
					</div>
					{errors.priority ? <span className='it-f__err'>{errors.priority}</span> : null}
				</div>

				<div className='it-ticket-grid'>
					<div className='it-f-label'>
						<label className='it-f-label' style={{gap: 4}}>
							<span className='it-f-label__text'>Заказчик</span>
							<select
								className={`it-f${errors.ownerId ? ' it-f--error' : ''}`}
								value={ownerId}
								onChange={e => {
									onChange({ownerId: e.target.value})
									onClearError('ownerId')
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
							{errors.ownerId ? <span className='it-f__err'>{errors.ownerId}</span> : null}
						</label>
					</div>

					<NotifierField
						options={notifierOptions}
						value={notifierIds}
						onChange={ids => onChange({notifierIds: ids})}
						pluginUserLabel={pluginUserLabel}
					/>

					{isManager && ownerId ? (
						<div style={{gridColumn: '1 / -1', marginTop: '-16px'}}>
							<label className='it-f-check' style={{display: 'flex'}}>
								<input
									className='it-f-check__box'
									type='checkbox'
									checked={ownerMuted}
									onChange={e => onChange({ownerMuted: e.target.checked})}
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
							onChange={e => onChange({groupId: e.target.value})}
							disabled={!isManager}
						>
							<option value=''>— Авто (по категории) —</option>
							{groups.map(g => (
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
							onChange={e => onChange({assigneeId: e.target.value})}
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
									checked={assigneeId === currentUserId}
									onChange={e => onChange({assigneeId: e.target.checked ? currentUserId : ''})}
								/>
								<span className='it-f-check__text'>Назначить меня исполнителем</span>
							</label>
						)}
					</label>

					<div style={{gridColumn: '1 / -1'}}>
						<label className='it-f-label'>
							<span className='it-f-label__text'>Срок выполнения</span>
							<input
								className='it-f'
								type='datetime-local'
								value={dueDate}
								onChange={e => onChange({dueDate: e.target.value})}
							/>
						</label>
					</div>
				</div>
			</div>
		</section>
	)
}
