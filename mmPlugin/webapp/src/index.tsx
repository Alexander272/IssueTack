import './index.css';

import {findBotDmChannelId, getCurrentTeamName, setStore} from './api';
import {installDeepLinkClickHandler, navigateToChannel, requestPluginOpen, setPendingOpen} from './deepLink';
import type {PluginChannel, PluginRegistry, PluginStore} from './types';
import appBarIcon from './assets/icon.png';
import ChannelHeaderIcon from './components/ChannelHeaderIcon';
import TicketDeepLinkPage from './components/TicketDeepLinkPage';

// openFromAppBar открывает модалку плагина по клику иконки в App Bar. Модалка
// живёт в личном диалоге с ботом (её хост — ChannelHeaderIcon, смонтированный
// только там), поэтому если мы уже в этом диалоге, просто просим открыть её
// на месте; иначе сначала уводим MM в диалог и оставляем флаг, по которому
// модалка откроется сама.
function openFromAppBar(currentChannelId: string | null): void {
    const botDmId = findBotDmChannelId();
    if (!botDmId || botDmId === currentChannelId) {
        requestPluginOpen();
        return;
    }
    const teamName = getCurrentTeamName();
    if (!teamName) {
        // Команда ещё не гидратирована — увести в канал нечем, открываем на месте.
        requestPluginOpen();
        return;
    }
    setPendingOpen();
    navigateToChannel(teamName, botDmId);
}

class IssuetrackPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void {
        setStore(store);
        registry.registerChannelHeaderIcon({component: ChannelHeaderIcon});
        // Иконка в правой колонке App Bar. Кнопка не «мёртвая»: она открывает ту
        // же модалку, что и иконка в шапке, уводя в диалог с ботом, если нужно.
        // Метода может не быть в старых версиях MM — тогда просто не регистрируем.
        if (typeof registry.registerAppBarComponent === 'function') {
            registry.registerAppBarComponent({
                iconUrl: appBarIcon,
                tooltipText: 'Заявки в IT отдел',
                action: (channel: PluginChannel) => openFromAppBar(channel ? channel.id : null),
            });
        }
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
