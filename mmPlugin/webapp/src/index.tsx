import './index.css';

import {setStore} from './api';
import type {PluginRegistry, PluginStore} from './types';
import ChannelHeaderIcon from './components/ChannelHeaderIcon';
import TicketDeepLinkPage from './components/TicketDeepLinkPage';

class IssuetrackPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void {
        setStore(store);
        registry.registerChannelHeaderIcon({component: ChannelHeaderIcon});
        // Mattermost монтирует маршрут по адресу /plug/issuetrack/ticket/<uuid>.
        // На этот адрес ведут ссылки из DM-уведомлений и подтверждения создания.
        registry.registerCustomRoute('/ticket/:id', TicketDeepLinkPage);
    }
}

window.registerPlugin('issuetrack', new IssuetrackPlugin());
