import type {ComponentType} from 'react';

export type TicketStatus = 'open' | 'in_progress' | 'pending' | 'on_hold' | 'resolved' | 'closed' | 'cancelled';

export type Priority = 'low' | 'medium' | 'high' | 'urgent';

export interface PluginUser {
    id: string;
    username: string;
    siteId?: string | null;
}

export interface PluginCategory {
    id: string;
    name: string;
    description: string;
    groupId: string;
    // Раздел категории (таксономия), заполняется сервером, если категория
    // привязана к разделу.
    categoryGroupId?: string | null;
    categoryGroup?: {id: string; name: string};
    priority: Priority;
    isActive: boolean;
    realmId: string;
    createdAt: string;
    updatedAt: string;
}

export interface PluginSite {
    id: string;
    name: string;
    address: string;
    createdAt: string;
    updatedAt: string;
}

export interface PluginGroupShort {
    id: string;
    name: string;
    // Дефолтный исполнитель группы — для веб-подобного автозаполнения: при
    // смене группы менеджеру подставляется defaultAssigneeId этой группы.
    defaultAssigneeId?: string | null;
}

// PluginScope — контекст вызова плагина: канал и Mattermost-пользователь.
// botUserId заполняется только для личного диалога с ботом реалма: такие каналы
// не привязаны к реалму, и сервер определяет его по собеседнику (проверяя состав
// участников канала).
export interface PluginScope {
    channelId: string;
    userId: string;
    botUserId?: string;
}

export interface PluginContextResult {
    bound: boolean;
    realmId: string;
    realmName: string;
    user: PluginUser;
    categories: PluginCategory[];
    sites: PluginSite[];
    // Роли и справочники формы создания. isManager (начальник области ИЛИ
    // управляет группой) открывает «Расширенные настройки», наличие групп
    // членства — секцию исполнителя «Заказчик». Зеркало веб-формы: условия
    // показа секций совпадают с web, финальные права всё равно у TicketService.Create.
    isManager: boolean;
    memberGroupIds: string[];
    groups: PluginGroupShort[];
    executors: PluginUserShort[];
    customers: PluginUserShort[];
}

export interface PluginTicketShort {
    id: string;
    number: number;
    title: string;
    description?: string;
    status: TicketStatus;
    priority: Priority;
    createdAt: string;
    category?: {id: string; name: string};
    site?: {id: string; name: string};
    link?: string;
}

export interface PluginUserShort {
    id: string;
    username: string;
    firstName: string;
    lastName: string;
    internalNumber: string;
}

export interface PluginAttachment {
    id: string;
    fileName: string;
    fileSize: number;
    mimeType: string;
}

export interface PluginTicketDetail {
    id: string;
    number: number;
    title: string;
    description: string;
    status: TicketStatus;
    priority: Priority;
    createdAt: string;
    dueDate?: string;
    category?: {id: string; name: string};
    site?: {id: string; name: string};
    creator?: PluginUserShort;
    owner?: PluginUserShort;
    assignee?: PluginUserShort;
    link?: string;
    deepLink?: string;
    attachments?: PluginAttachment[];
    canConfirm?: boolean;
    canReopen?: boolean;
    canCancel?: boolean;
}

// PluginTicketLinkContext — ответ для страницы заявки, открытой по deep-link.
// На этой странице нет текущего канала, поэтому channelId приходит с сервера.
export interface PluginTicketLinkContext {
    channelId: string;
    detail: PluginTicketDetail;
}

export interface PluginCommentUser {
    id: string;
    name: string;
}

export interface PluginCommentAttachment {
    id: string;
    fileName: string;
    fileSize: number;
    mimeType: string;
}

export interface PluginComment {
    id: string;
    text: string;
    createdAt: string;
    user?: PluginCommentUser;
    attachments?: PluginCommentAttachment[];
}

export interface PluginCreateResult {
    id: string;
    number?: number;
    title: string;
    link?: string;
    deepLink?: string;
}

export interface PluginApiErrorBody {
    id?: unknown;
    code?: string;
    message?: string;
}

export interface PluginStore {
    getState: () => {
        entities: {
            users: {
                currentUserId?: string;
            };
            // Имя текущей команды нужно, чтобы увести ссылку с маршрута плагина в
            // канал: маршрут плагина живёт вне команд, а адрес канала — /{team}/channels/{id}.
            teams: {
                currentTeamId?: string;
                teams: Record<string, {id: string; name: string}>;
            };
        };
    };
    subscribe: (listener: () => void) => () => void;
}

export interface PluginRegistry {
    registerChannelHeaderIcon(input: {component: ComponentType<{channel: {id: string}}>}): string;

    // registerCustomRoute монтирует компонент по адресу
    // /plug/<pluginId>/<route>. Компонент не получает props (в том числе
    // match.params), поэтому параметры маршрута парсятся из location.pathname.
    registerCustomRoute(route: string, component: ComponentType): string;
}

export interface IssuetrackWebappPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void;
}