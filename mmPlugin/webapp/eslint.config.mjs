// У плагина свой package.json без eslint-зависимостей. Переиспользуем конфиг
// и уже установленный ESLint из frontend, чтобы правила линта не разъезжались.
import frontendConfig from '../../frontend/eslint.config.js'

export default [
	{ ignores: ['dist/**', 'node_modules/**'] },
	...frontendConfig,
	{
		// Плагин живёт внутри Mattermost и синхронизирует состояние с его сторам
		// и deep-link'ами в эффектах. Это устоявшийся код, а не
		// новая форма: правило React Compiler здесь не применимо без переделки
		// шины открытия модалки, поэтому для плагина его отключаем.
		rules: {
			'react-hooks/set-state-in-effect': 'off',
		},
	},
]
