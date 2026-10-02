import './index.css';

import {setStore} from './api';
import {installDeepLinkClickHandler} from './deepLink';
import type {PluginRegistry, PluginStore} from './types';
import ChannelHeaderIcon from './components/ChannelHeaderIcon';
import TicketDeepLinkPage from './components/TicketDeepLinkPage';

class IssuetrackPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void {
        setStore(store);
        registry.registerChannelHeaderIcon({component: ChannelHeaderIcon});
        // Mattermost монтирует маршрут по адресу /plug/issuetrack/ticket/<uuid>.
        // На этот адрес ведут ссылки из DM-уведомлений и подтверждения создания.
        // Обычно по такой ссылке заявка открывается модалкой поверх чата, без
        // перехода: клик перехватывается, если модалка уже смонтирована. Сюда
        // попадаем, когда её нет — новая вкладка или чужой канал.
        registry.registerCustomRoute('/ticket/:id', TicketDeepLinkPage);
        installDeepLinkClickHandler();
    }
}

window.registerPlugin('issuetrack', new IssuetrackPlugin());
