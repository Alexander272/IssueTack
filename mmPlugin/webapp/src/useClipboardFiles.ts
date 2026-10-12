import { useEffect, useRef } from 'react'

// Элементы плагина в DOM Mattermost: модалка поверх чата (.it-ticket-modal) и
// отдельная страница по deep-link (.it-ticket-page).
const PLUGIN_MODAL_SELECTOR = '.it-ticket-modal'
const PLUGIN_ROOT_SELECTOR = '.it-ticket-modal, .it-ticket-page'

// clipboardFiles достаёт файлы из буфера обмена: скриншот или файл,
// скопированный в проводнике. Опираться на navigator.clipboard нельзя — на LAN
// без TLS это не secure context и API там нет, а clipboardData внутри события
// paste доступен всегда, поэтому Ctrl+V работает и на http.
export function clipboardFiles(e: ClipboardEvent): File[] | null {
	const dt = e.clipboardData
	if (!dt) {
		return null
	}
	if (dt.files && dt.files.length > 0) {
		return Array.from(dt.files)
	}
	const fromItems = Array.from(dt.items || [])
		.filter(item => item.kind === 'file')
		.map(item => item.getAsFile())
		.filter((f): f is File => f !== null)
	return fromItems.length > 0 ? fromItems : null
}

// ignorePaste — пропустить ли вставку, не трогая её. Пока открыта модалка
// плагина, поле сообщения Mattermost за подложкой недоступно, поэтому вставку с
// файлом всегда направляем в плагин (фокус мог остаться в composer — событие
// paste приходит именно тому элементу, который в фокусе). Гард по «чужому»
// редактируемому полю нужен лишь когда модалки нет (страница по deep-link),
// чтобы не перехватывать вставку в чужие поля.
function ignorePaste(target: EventTarget | null): boolean {
	if (typeof document !== 'undefined' && document.querySelector(PLUGIN_MODAL_SELECTOR)) {
		return false
	}
	if (!(target instanceof HTMLElement)) {
		return false
	}
	const editable = target.closest('input, textarea, [contenteditable=""], [contenteditable="true"]')
	return !!editable && !editable.closest(PLUGIN_ROOT_SELECTOR)
}

// useClipboardFiles вешает слушатель paste и отдаёт наружу файлы из буфера, если
// они там есть. Текстовая вставка не перехватывается: если в буфере нет файла,
// событие не трогаем и оно доходит до поля как обычно. Слушатель на window, а не
// на конкретном узле: внутри модалки фокус может стоять на body.
export function useClipboardFiles(onFiles: (files: File[]) => void): void {
	const handlerRef = useRef(onFiles)
	useEffect(() => {
		handlerRef.current = onFiles
	})
	useEffect(() => {
		const onPaste = (e: ClipboardEvent) => {
			if (ignorePaste(e.target)) {
				return
			}
			const files = clipboardFiles(e)
			if (!files) {
				return
			}
			// Не только preventDefault, но и stopImmediatePropagation: без него
			// файл из буфера попадает ещё и в чат Mattermost — модалка плагина
			// порталится в document.body, поэтому событие всё равно доходит до
			// composer за подложкой, а свой обработчик paste MM не сверяется с
			// defaultPrevented.
			e.preventDefault()
			e.stopImmediatePropagation()
			handlerRef.current(files)
		}
		// capture-фаза: событие paste получаем раньше цели, поэтому вставку не
		// может «съесть» чужой обработчик composer MM.
		window.addEventListener('paste', onPaste, true)
		return () => window.removeEventListener('paste', onPaste, true)
	}, [])
}
