import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { registerSW } from 'virtual:pwa-register'
import { setGlobalDevModeChecks } from 'reselect'

import App from './App'

if (import.meta.env.DEV) {
	// Гасим dev-only предупреждение reselect "An input selector returned a different
	// result...", которое прилетает из нестабильного селектора внутри @mui/x-charts
	// (useAxesTooltip). Это апстрим-баг библиотеки, к нашему коду отношения не имеет.
	setGlobalDevModeChecks({ inputStabilityCheck: 'never' })
}

registerSW({
	immediate: true,
	onRegisteredSW(swScriptUrl) {
		console.log('SW registered: ', swScriptUrl)
	},
	onRegisterError(error) {
		console.log('SW registration error', error)
	},
})

createRoot(document.getElementById('root')!).render(
	<StrictMode>
		<App />
	</StrictMode>,
)
