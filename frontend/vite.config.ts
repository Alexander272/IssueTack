import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
	plugins: [
		react({
			plugins: [['@swc/plugin-emotion', {}]],
		}),
		VitePWA({
			injectRegister: 'auto',
			registerType: 'autoUpdate',
			workbox: {
				globPatterns: ['**/*.{js,css,html,ico,png,jpg,jpeg,webp,svg,woff,woff2}'],
			},
			manifest: {
				id: 'identic',
				name: 'Identic',
				short_name: 'Identic',
				description: 'Журнал',
				lang: 'ru',
				theme_color: '#fafafa',
				background_color: '#fafafa',
				icons: [
					{
						src: 'favicon.ico',
						type: 'image/x-icon',
						sizes: '100x97',
					},
					{
						src: 'logo192.webp',
						type: 'image/webp',
						sizes: '192x192',
					},
					{
						src: 'logo512.webp',
						type: 'image/webp',
						sizes: '512x512',
					},
				],
			},
		}),
	],
	resolve: {
		alias: [
			{
				find: '@',
				replacement: path.resolve(__dirname, 'src'),
			},
		],
	},
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:9000',
				changeOrigin: true,
				ws: true, // Включает проксирование веб-сокетов
			},
		},
	},
	build: {
		target: 'es2021',
	},
})
