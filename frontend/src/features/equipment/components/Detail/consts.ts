// Подпись секции вкладки «Обзор» — как в макете: мелкая, в верхнем регистре,
// приглушённая. Файл без компонентов: константу рядом с компонентом не держим
// (react-refresh/only-export-components).
export const sectionLabelSx = {
	fontSize: 12,
	fontWeight: 700,
	letterSpacing: '0.06em',
	textTransform: 'uppercase' as const,
	color: '#94a3b8',
}

// Второстепенная кнопка секции (мягкая синяя) — как в макете: «Изменить состав»,
// «Выдать» и «Возврат».
export const softButtonSx = {
	borderRadius: '10px',
	textTransform: 'none' as const,
	fontSize: 12,
	fontWeight: 600,
	px: 1.5,
	py: 0.5,
	minWidth: 0,
	bgcolor: '#eff6ff',
	color: '#2563eb',
	'&:hover': { bgcolor: '#dbeafe' },
}
