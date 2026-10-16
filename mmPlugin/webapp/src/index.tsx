import './index.css'

import { setStore } from './api'
import { installDeepLinkClickHandler, requestPluginOpen } from './deepLink'
import type { PluginRegistry, PluginStore } from './types'
import appBarIcon from './assets/icon.jpg'
import ChannelHeaderIcon from './components/ChannelHeaderIcon'
import TicketDeepLinkPage from './components/TicketDeepLinkPage'

class IssuetrackPlugin {
	initialize(registry: PluginRegistry, store: PluginStore): void {
		setStore(store)
		registry.registerChannelHeaderIcon({ component: ChannelHeaderIcon })
		// Иконка в правой колонке App Bar открывает ту же модалку, что и иконка в
		// шапке, но из любого канала и без перехода: модалку хостит
		// ChannelHeaderIcon, смонтированный в шапке каждого канала, а нужный
		// DM-канал бота он подтягивает с бэкенда. Метода может не быть в старых
		// версиях MM — тогда просто не регистрируем иконку.
		if (typeof registry.registerAppBarComponent === 'function') {
			registry.registerAppBarComponent({
				iconUrl: appBarIcon,
				tooltipText: 'Заявки в IT отдел',
				action: () => requestPluginOpen(),
			})
		}
		// Mattermost монтирует маршрут по адресу /plug/issuetrack/ticket/<uuid>.
		// На этот адрес ведут ссылки из DM-уведомлений и подтверждения создания.
		// Обычно по такой ссылке заявка открывается модалкой поверх чата, без
		// перехода: клик перехватывается, если модалка уже смонтирована. Сюда
		// попадаем, когда её нет — новая вкладка или чужой канал.
		registry.registerCustomRoute('/ticket/:id', TicketDeepLinkPage)
		installDeepLinkClickHandler()
	}
}

window.registerPlugin('issuetrack', new IssuetrackPlugin())
