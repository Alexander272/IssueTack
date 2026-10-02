// deepLink.ts — открытие заявки из ссылки без навигации.
//
// Ссылка из DM ведёт на маршрут плагина /plug/issuetrack/ticket/<uuid>, где
// Mattermost монтирует наш компонент как единственный вид в .main-wrapper: там
// нет ни чата, ни меню каналов, ни хоста модалки. Но клик по ссылке происходит
// в обычном MM — в DM с ботом, где модалка уже смонтирована (её открывает
// иконка в шапке). Поэтому клик перехватывается, и заявка показывается поверх
// чата без перехода вообще: ни перерисовки вида, ни перезагрузки.
//
// Если модалка не смонтирована (канал не тот или новая вкладка), клик не
// трогаем — тогда MM сам уходит на маршрут, и заявку показывает страница
// TicketDeepLinkPage. То же самое делает и средняя кнопка: ссылки в «открыть в
// новой вкладке» остаются рабочими.
//
// Ниже — сигнал через sessionStorage для этого запасного пути: переход туда
// делается клиентской навигацией, а при сбое через location.assign, и после
// полной перезагрузки модульные переменные уже пусты. Все take* читают и
// удаляют запись — сигнал одноразовый.

const CHANNEL_KEY = 'issuetrack.deeplink.channel';
const TICKET_KEY = 'issuetrack.deeplink.ticket';

// Заявка, ждавшая перехода дольше минуты, считается протухшей: иначе ссылка,
// открытая в фоне, могла бы через час открыть модалку в случайном канале.
const TICKET_TTL_MS = 60_000;

// Полный маркер ссылки, в отличие от ROUTE_MARKER: в href нужно отсечь и
// /plug, иначе подцепим чужой путь, у которого случайно совпал хвост.
const DEEP_LINK_MARKER = '/plug/issuetrack/ticket/';

// Маркер маршрута плагина в pathname. По нему же определяется база установки:
// путь может быть /plug/issuetrack/ticket/<uuid> или, при установке под
// субпатом, /mattermost/plug/issuetrack/ticket/<uuid>.
const ROUTE_MARKER = '/issuetrack/ticket/';

const TICKET_ID_RE = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/

function readItem(key: string): string | null {
    try {
        return window.sessionStorage.getItem(key);
    } catch (e) {
        return null;
    }
}

function writeItem(key: string, value: string): void {
    try {
        window.sessionStorage.setItem(key, value);
    } catch (e) {
        // приватный режим или переполнение — сигнал просто не долетит,
        // обработчик ссылка откроет как обычную страницу заявки
    }
}

function removeItem(key: string): void {
    try {
        window.sessionStorage.removeItem(key);
    } catch (e) {
        // см. writeItem
    }
}

export interface RememberedChannel {
    teamName: string;
    channelId: string;
}

// rememberChannel — запоминает канал, из которого открыли ссылку, вместе с
// командой: адрес канала начинается с имени команды, а текущая команда к моменту
// клика могла смениться. Вызывается иконкой в шапке канала, поэтому к моменту
// клика данные уже есть.
export function rememberChannel(teamName: string, channelId: string): void {
    if (channelId) {
        writeItem(CHANNEL_KEY, JSON.stringify({teamName, channelId}));
    }
}

export function takeChannel(): RememberedChannel | null {
    const raw = readItem(CHANNEL_KEY);
    if (!raw) {
        return null;
    }
    removeItem(CHANNEL_KEY);
    try {
        const parsed = JSON.parse(raw) as Partial<RememberedChannel>;
        return parsed.teamName && parsed.channelId ?
            {teamName: parsed.teamName, channelId: parsed.channelId} :
            null;
    } catch (e) {
        return null;
    }
}

// restoreChannel возвращает запись, забранную takeChannel, когда уйти в канал не
// вышло (например, имя команды ещё не гидратировано в сторе). Иначе канал терялся
// бы до конца сессии, и вторая ссылка открылась бы страницей, хотя первая уже
// показала, что канал есть.
export function restoreChannel(channel: RememberedChannel): void {
    rememberChannel(channel.teamName, channel.channelId);
}

export function setPendingTicket(ticketId: string): void {
    writeItem(TICKET_KEY, JSON.stringify({id: ticketId, at: Date.now()}));
}

export function takePendingTicket(): string | null {
    const raw = readItem(TICKET_KEY);
    if (!raw) {
        return null;
    }
    removeItem(TICKET_KEY);
    try {
        const parsed = JSON.parse(raw) as {id?: string; at?: number};
        if (!parsed.id || typeof parsed.at !== 'number') {
            return null;
        }
        return Date.now() - parsed.at > TICKET_TTL_MS ? null : parsed.id;
    } catch (e) {
        return null;
    }
}

// channelPath — адрес канала относительно текущей установки: база отрезается от
// pathname по маркеру маршрута плагина, чтобы не сломаться на субпате
// (/mattermost/plug/…), где жёсткий префикс / дал бы /mattermost/it/..., если
// MM стоит в корне, и наоборот потерял бы субпат.
export function channelPath(teamName: string, channelId: string): string | null {
    if (!teamName || !channelId) {
        return null;
    }
    const path = window.location.pathname || '';
    const at = path.indexOf(ROUTE_MARKER);
    // Отрезаем маршрут плагина и его префикс /plug: база установки — это то, что
    // стоит перед ними ('' в корне, '/mattermost' под субпатом).
    const base = (at === -1 ? path : path.slice(0, at)).replace(/\/plug$/, '').replace(/\/+$/, '');
    return `${base}/${encodeURIComponent(teamName)}/channels/${encodeURIComponent(channelId)}`;
}

// navigateClientSide — уход в канал без перезагрузки Mattermost. В отличие от
// location.assign полной перезагрузкой (медленно, но работает всегда) это
// оставляет приложение живым: MM перерисовывает вид канала сам. Работает через
// pushState + синтетический popstate, который подхватывает react-router.
//
// Если переход не вышел, вызывающий это заметит сам: компонент останется
// смонтированным, и он уйдёт на reloadTo.
export function navigateClientSide(path: string): void {
    window.history.pushState(window.history.state, '', path);
    window.dispatchEvent(new PopStateEvent('popstate', {state: window.history.state}));
}

export function reloadTo(path: string): void {
    window.location.assign(path);
}

// Разбор ссылки на заявку из href. Берём атрибут, а не anchor.href: там путь
// остаётся site-relative (/plug/issuetrack/ticket/<uuid>), и мы не зависим от
// того, как Mattermost смонтирован. Установка под субпатом не мешает — маркер
// ищется внутри строки, а не сравнением с pathname целиком.
export function matchDeepLink(href: string): string | null {
    const path = href.split('#')[0].split('?')[0];
    const at = path.indexOf(DEEP_LINK_MARKER);
    if (at === -1) {
        return null;
    }
    const raw = (path.slice(at + DEEP_LINK_MARKER.length).split('/')[0] || '');
    const decoded = (() => {
        try {
            return decodeURIComponent(raw);
        } catch (e) {
            return raw;
        }
    })();
    return TICKET_ID_RE.test(decoded) ? decoded : null;
}

// Хост модалки: есть ли сейчас смонтированный ChannelHeaderIcon, готовый
// показать заявку. Ставится иконкой в шапке, а проверяется перехватчиком клика:
// перехватывать навигацию имеет смысл только когда есть куда показать заявку.
let modalHostPresent = false;

export function setModalHost(present: boolean): void {
    modalHostPresent = present;
}

export function isModalHost(): boolean {
    return modalHostPresent;
}

type OpenListener = (ticketId: string) => void;

const openListeners = new Set<OpenListener>();

export function subscribeOpen(listener: OpenListener): () => void {
    openListeners.add(listener);
    return () => {
        openListeners.delete(listener);
    };
}

export function requestOpen(ticketId: string): void {
    openListeners.forEach(listener => listener(ticketId));
}

// installDeepLinkClickHandler перехватывает клик по ссылке на заявку, когда
// модалка уже смонтирована, и показывает заявку поверх чата вместо перехода на
// маршрут плагина. Никакой навигации: смена вида и перезагрузка MM исчезают
// вместе с маршрутом.
//
// Перехват на document в фазе capture и до перехода на target: только так мы
// опережаем обработчик клика самого Mattermost. Гасим событие через
// preventDefault + stopPropagation — у stopPropagation() есть смысл, потому
// что не вся обработка ссылок в MM проверяет defaultPrevented и иначе
// переход всё равно случился бы.
//
// Что намеренно не трогаем: среднюю кнопку, Ctrl/Shift/Alt (открыть в новой
// вкладке/окне) и ссылки с target/download — это нативное поведение браузера,
// и в новой вкладке заявку открывает страница.
export function installDeepLinkClickHandler(): void {
    if (typeof document === 'undefined' || typeof document.addEventListener !== 'function') {
        return;
    }
    document.addEventListener('click', (event: Event) => {
        const mouseEvent = event as MouseEvent;
        if (mouseEvent.defaultPrevented || mouseEvent.button !== 0) {
            return;
        }
        if (mouseEvent.metaKey || mouseEvent.ctrlKey || mouseEvent.shiftKey || mouseEvent.altKey) {
            return;
        }
        const origin = event.target;
        const anchor = origin instanceof Element ? origin.closest('a') : null;
        if (!anchor || anchor.hasAttribute('download')) {
            return;
        }
        const anchorTarget = anchor.getAttribute('target');
        if (anchorTarget && anchorTarget !== '_self') {
            return;
        }
        const href = anchor.getAttribute('href');
        const ticketId = href ? matchDeepLink(href) : null;
        if (!ticketId || !isModalHost()) {
            return;
        }
        mouseEvent.preventDefault();
        mouseEvent.stopPropagation();
        requestOpen(ticketId);
    }, true);
}
