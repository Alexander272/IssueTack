import { useState, useRef, useEffect, type FC } from 'react'
import { FormControl, type SxProps, type Theme } from '@mui/material'

type Props = {
	value: boolean
	onChange: (value: boolean) => void
	labels?: [string, string]
	sx?: SxProps<Theme>
}

export const Switch: FC<Props> = ({ value, onChange, labels = ['Нет', 'Да'], sx }) => {
	const containerRef = useRef<HTMLDivElement>(null)
	const [width, setWidth] = useState(0)

	useEffect(() => {
		if (containerRef.current) {
			const buttons = containerRef.current.querySelectorAll('.switch-btn')
			if (buttons[0]) {
				setWidth((buttons[0] as HTMLElement).offsetWidth)
			}
		}
	}, [])

	return (
		<FormControl
			fullWidth
			sx={{
				position: 'relative',
				backgroundColor: '#f8fafc',
				border: '1px solid #e2e8f0',
				borderRadius: '20px',
				padding: '2px',
				height: 36,
				...sx,

				'--current-radius': (theme: Theme) => {
					// Извлекаем borderRadius из текущего объекта (который уже объединил дефолт и пропсы)
					// Мы берем его из sx (если он там есть) или оставляем наш дефолт '20px'
					// eslint-disable-next-line @typescript-eslint/no-explicit-any
					const finalRadius = (sx as any)?.borderRadius ?? '20px'

					// Если это число (например, borderRadius: 2), MUI умножает его на theme.shape.borderRadius (обычно 4px)
					if (typeof finalRadius === 'number') {
						return `${(theme.shape.borderRadius as number) * finalRadius}px`
					}
					return finalRadius
				},
			}}
		>
			<div
				ref={containerRef}
				style={{
					position: 'relative',
					display: 'flex',
					height: '100%',
					width: '100%',
					zIndex: 2,
				}}
			>
				<div
					className='switch-slider'
					style={{
						position: 'absolute',
						top: 2,
						left: 2,
						width: `calc(50% - 2px)`,
						height: 'calc(100% - 4px)',
						backgroundColor: value ? '#0432a5' : '#ffffff',
						borderRadius: 'calc(var(--current-radius, 22px) - 2px)',
						boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
						transition:
							'transform 0.2s cubic-bezier(0.4, 0, 0.2, 1), background-color 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
						transform: value ? `translateX(${width}px)` : 'translateX(0)',
						zIndex: 1,
					}}
				/>
				<button
					type='button'
					className='switch-btn'
					onClick={() => onChange(false)}
					style={{
						flex: 1,
						border: 'none',
						background: 'transparent',
						borderRadius: '20px',
						color: !value ? '#0f172a' : '#64748b',
						fontSize: 13,
						fontWeight: !value ? 600 : 500,
						cursor: 'pointer',
						transition: 'color 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
						position: 'relative',
						zIndex: 2,
					}}
				>
					{labels[0]}
				</button>
				<button
					type='button'
					className='switch-btn'
					onClick={() => onChange(true)}
					style={{
						flex: 1,
						border: 'none',
						background: 'transparent',
						borderRadius: '20px',
						color: value ? '#ffffff' : '#64748b',
						fontSize: 13,
						fontWeight: value ? 500 : 500,
						cursor: 'pointer',
						transition: 'color 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
						position: 'relative',
						zIndex: 2,
					}}
				>
					{labels[1]}
				</button>
			</div>
		</FormControl>
	)
}
