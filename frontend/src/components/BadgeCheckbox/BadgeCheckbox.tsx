import { type FC, useId } from 'react'
import { styled } from '@mui/material'
import { CheckIcon, XIcon } from 'lucide-mui'

const Input = styled('input')({
	position: 'absolute',
	opacity: 0,
	width: 0,
	height: 0,
	margin: 0,
})

const Label = styled('label')<{ checked: boolean; disabled: boolean }>(({ checked, disabled }) => ({
	position: 'relative',
	display: 'inline-flex',
	alignItems: 'center',
	justifyContent: 'center',
	width: 30,
	height: 30,
	borderRadius: 10,
	fontSize: 18,
	flexShrink: 0,
	border: '2px solid transparent',
	transition: 'all 0.2s cubic-bezier(0.2,0,0,1)',
	cursor: disabled ? 'not-allowed' : 'pointer',
	opacity: disabled ? 0.5 : 1,
	filter: disabled ? 'grayscale(0.5)' : 'none',
	userSelect: 'none',

	...(checked
		? {
				background: '#dbeafe',
				color: '#2563eb',
				borderColor: '#93c5fd',
				boxShadow: '0 4px 12px rgba(37,99,235,0.15)',
			}
		: {
				background: '#f1f5f9',
				color: '#cbd5e1',
				borderColor: '#e2e8f0',
			}),
}))

interface BadgeCheckboxProps {
	checked: boolean
	disabled?: boolean
	onChange: () => void
	ariaLabel?: string
}

export const BadgeCheckbox: FC<BadgeCheckboxProps> = ({ checked, disabled = false, onChange, ariaLabel }) => {
	const inputId = useId()
	return (
		<Label checked={checked} disabled={disabled}>
			<Input
				id={inputId}
				type='checkbox'
				checked={checked}
				disabled={disabled}
				onChange={onChange}
				aria-label={ariaLabel}
			/>
			{checked ? <CheckIcon sx={{ fontSize: 18 }} /> : <XIcon sx={{ fontSize: 18 }} />}
		</Label>
	)
}
