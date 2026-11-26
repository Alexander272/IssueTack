// Секция 3 для «чистого» заявителя: только дополнительные уведомления.
// У менеджера/исполнителя вместо неё стоит AdvancedSection.

import React from 'react'

import type {PluginUserShort} from '../../types'
import {NotifierField} from '../Form/NotifierField'
import {pluginUserLabel} from './model'

interface NotifierSectionProps {
	options: PluginUserShort[]
	value: string[]
	onChange: (ids: string[]) => void
}

export function NotifierSection({options, value, onChange}: NotifierSectionProps) {
	return (
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
					options={options}
					value={value}
					onChange={onChange}
					pluginUserLabel={pluginUserLabel}
					label={null}
				/>
			</div>
		</section>
	)
}
