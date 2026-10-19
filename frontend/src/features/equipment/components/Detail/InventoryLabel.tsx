import { useEffect, useRef, type FC } from 'react'
import { createPortal } from 'react-dom'
import { Box, GlobalStyles, Typography } from '@mui/material'
import JsBarcode from 'jsbarcode'

import type { IEquipment } from '../../types/equipment'

// Штрихкод Code128 рисуем вектором (SVG): этикетка печатается резко и без растра.
// JsBarcode работает императивно и с реальным DOM-узлом, поэтому здесь эффект и
// ref, а не декларативный рендер. Размеры: 2px на модуль × 50px высоты — этого
// хватает сканеру и на экране, и на бумаге.
export const InventoryBarcode: FC<{ value: string; height?: number }> = ({ value, height = 50 }) => {
	const ref = useRef<SVGSVGElement>(null)

	useEffect(() => {
		if (!ref.current) return
		try {
			JsBarcode(ref.current, value, {
				format: 'CODE128',
				displayValue: false,
				width: 2,
				height,
				margin: 0,
				background: '#ffffff',
				lineColor: '#000000',
			})
		} catch {
			// Значение не кодируется (непечатаемые символы) — оставляем только номер
			// текстом: пользователь видит, что штрихкод не построился, а не пустоту.
		}
	}, [value, height])

	return <svg ref={ref} style={{ maxWidth: '100%', height: 'auto' }} />
}

type LabelProps = {
	name: string
	inventoryNumber: string
	// print — размеры этикетки для бумаги (мм), иначе превью в карточке.
	print?: boolean
}

// InventoryLabel — содержимое этикетки: наименование, штрихкод и номер. Один вид
// и для превью в карточке, и для печати, чтобы «что вижу, то и печатаю».
export const InventoryLabel: FC<LabelProps> = ({ name, inventoryNumber, print }) => (
	<Box
		sx={{
			width: print ? '80mm' : '100%',
			maxWidth: '100%',
			border: '1px dashed #cbd5e1',
			borderRadius: '2mm',
			bgcolor: '#fff',
			p: print ? '3mm' : 2,
			display: 'flex',
			flexDirection: 'column',
			alignItems: 'center',
			gap: print ? '1.5mm' : 1,
		}}
	>
		<Typography
			sx={{
				fontSize: print ? '4mm' : 14,
				fontWeight: 600,
				lineHeight: 1.2,
				textAlign: 'center',
				color: '#0f172a',
				overflow: 'hidden',
				display: '-webkit-box',
				WebkitLineClamp: 2,
				WebkitBoxOrient: 'vertical',
				wordBreak: 'break-word',
			}}
		>
			{name}
		</Typography>
		<InventoryBarcode value={inventoryNumber} height={print ? 60 : 50} />
		<Typography sx={{ fontFamily: 'monospace', fontSize: print ? '4mm' : 13, color: '#334155', letterSpacing: 1 }}>
			{inventoryNumber}
		</Typography>
	</Box>
)

// InventoryLabelPrint печатает одну этикетку на листе A4. Содержимое уходит в
// портал к body и видно только в @media print: на экране слой не мешает, а после
// печати (даже если браузер не прислал afterprint) ничего не «залипает».
export const InventoryLabelPrint: FC<{ unit: IEquipment; onClose: () => void }> = ({ unit, onClose }) => {
	useEffect(() => {
		const handleAfterPrint = () => onClose()
		window.addEventListener('afterprint', handleAfterPrint)
		// Ждём кадр: дочерний эффект InventoryBarcode рисует SVG раньше, но даём
		// браузеру пересчитать layout перед открытием диалога печати.
		const raf = requestAnimationFrame(() => window.print())
		return () => {
			cancelAnimationFrame(raf)
			window.removeEventListener('afterprint', handleAfterPrint)
		}
	}, [onClose])

	return createPortal(
		<>
			<GlobalStyles
				styles={{
					'@page': { size: 'A4', margin: '12mm' },
					'@media print': {
						// Печатаем только этикетку: остальные прямые дети body (#root,
						// тосты, порталы диалогов) в печать не попадают.
						'body > *:not(.inventory-print-root)': { display: 'none !important' },
						'.inventory-print-root': { display: 'block !important' },
					},
				}}
			/>
			<Box className='inventory-print-root' sx={{ display: 'none' }}>
				<InventoryLabel name={unit.name} inventoryNumber={unit.inventoryNumber ?? ''} print />
			</Box>
		</>,
		document.body,
	)
}
