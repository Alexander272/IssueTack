import type {
    PluginApiErrorBody,
    PluginComment,
    PluginContextResult,
    PluginCreateResult,
    PluginScope,
    PluginStore,
    PluginTicketDetail,
    PluginTicketLinkContext,
    PluginTicketShort,
} from './types';

let storeRef: PluginStore | null = null;

export function setStore(store: PluginStore): void {
    storeRef = store;
}

export function getCurrentUserId(): string | null {
    if (!storeRef || typeof storeRef.getState !== 'function') {
        return null;
    }
    const state = storeRef.getState();
    return state && state.entities && state.entities.users ?
        state.entities.users.currentUserId ?? null :
        null;
}

// getCurrentTeamName — имя команды текущего пользователя. Нужно для адреса
// канала: маршрут плагина живёт вне команд, а канал открывается как
// /{team}/channels/{id}, и team мы знаем только из стора.
export function getCurrentTeamName(): string | null {
    if (!storeRef || typeof storeRef.getState !== 'function') {
        return null;
    }
    const state = storeRef.getState();
    const teams = state && state.entities ? state.entities.teams : undefined;
    if (!teams || !teams.teams) {
        return null;
    }
    const current = teams.currentTeamId ? teams.teams[teams.currentTeamId] : undefined;
    return current && current.name ? current.name : null;
}

export function subscribeCurrentUser(listener: (userId: string | null) => void): () => void {
    if (!storeRef || typeof storeRef.subscribe !== 'function') {
        return () => undefined;
    }
    let last = getCurrentUserId();
    return storeRef.subscribe(() => {
        const next = getCurrentUserId();
        if (next !== last) {
            last = next;
            listener(next);
        }
    });
}

export class ApiError extends Error {
    status: number;

    code: string;

    constructor(status: number, code: string, message: string) {
        super(message || `Ошибка HTTP ${status}`);
        this.name = 'ApiError';
        this.status = status;
        this.code = code;
    }
}

function apiHeaders(custom?: Record<string, string>): Record<string, string> {
    const headers: Record<string, string> = {'X-Requested-With': 'XMLHttpRequest'};
    let csrf: string | null = null;
    try {
        csrf = window.localStorage.getItem('csrfToken');
    } catch (e) {
        // ignore localStorage access errors
    }
    if (csrf) {
        headers['X-CSRF-Token'] = csrf;
    }
    return {...headers, ...(custom || {})};
}

interface RequestOptions {
    method?: string;
    headers?: Record<string, string>;
    body?: BodyInit;
}

async function request<T>(path: string, options?: RequestOptions): Promise<T> {
    let res: Response;
    try {
        res = await fetch(path, {
            credentials: 'same-origin',
            headers: apiHeaders(options ? options.headers : undefined),
            method: options ? options.method : undefined,
            body: options ? options.body : undefined,
        });
    } catch (e) {
        throw new ApiError(0, 'NETWORK', 'Не удалось связаться с сервером');
    }

    let body: PluginApiErrorBody & {data?: T} | null = null;
    try {
        body = await res.json();
    } catch (e) {
        body = null;
    }

    if (!res.ok) {
        const message = body && body.message ? body.message : `Ошибка HTTP ${res.status}`;
        throw new ApiError(res.status, body && body.code ? body.code : 'UNKNOWN', message);
    }

    return body ? body.data as T : undefined as T;
}

export interface CreateTicketPayload {
    title: string;
    description?: string;
    categoryId?: string | null;
    siteId?: string | null;
    files: File[];
    // Доп. секции формы (менеджер/исполнитель). Сервер сам применяет ролевые
    // ограничения: для пользователя без полных прав группа/приоритет всё равно
    // берутся из категории, заказчик обязателен, срок отбрасывается.
    priority?: string;
    groupId?: string;
    assigneeId?: string;
    ownerId?: string;
    dueDate?: string;
}

const contextCache = new Map<string, {scope: PluginScope; ts: number; data: PluginContextResult}>();
// contextInflight дедуплицирует параллельные запросы /context одного канала:
// шапка канала может ре-рендериться несколько раз, пока ответ ещё в пути.
const contextInflight = new Map<string, Promise<PluginContextResult>>();
const CONTEXT_TTL_MS = 5 * 60 * 1000;

// scopeQuery собирает query-параметры scope. botUserId уходит только для личного
// диалога — на привязанных каналах сервер решает по channelId.
function scopeQuery(scope: PluginScope): URLSearchParams {
    const qs = new URLSearchParams({channelId: scope.channelId, userId: scope.userId});
    if (scope.botUserId) {
        qs.set('botUserId', scope.botUserId);
    }
    return qs;
}

export function getCachedContext(scope: PluginScope | null): PluginContextResult | null {
    if (!scope || !scope.channelId || !scope.userId) {
        return null;
    }
    const cached = contextCache.get(scope.channelId);
    if (cached && cached.scope.userId === scope.userId && Date.now() - cached.ts < CONTEXT_TTL_MS) {
        return cached.data;
    }
    return null;
}

export async function getContext(scope: PluginScope): Promise<PluginContextResult> {
    const cached = getCachedContext(scope);
    if (cached) {
        return cached;
    }
    const inflight = contextInflight.get(scope.channelId);
    if (inflight) {
        return inflight;
    }
    const pending = (async () => {
        const qs = scopeQuery(scope);
        const data = await request<PluginContextResult>(`/plugins/issuetrack/api/context?${qs.toString()}`, {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({channelId: scope.channelId, userId: scope.userId, botUserId: scope.botUserId || ''}),
        });
        contextCache.set(scope.channelId, {scope, ts: Date.now(), data});
        return data;
    })();
    contextInflight.set(scope.channelId, pending);
    try {
        return await pending;
    } finally {
        contextInflight.delete(scope.channelId);
    }
}

export async function getContextFresh(scope: PluginScope): Promise<PluginContextResult> {
    contextCache.delete(scope.channelId);
    return getContext(scope);
}

export async function getMyTickets(scope: PluginScope): Promise<PluginTicketShort[]> {
    return request<PluginTicketShort[]>(`/plugins/issuetrack/api/tickets?${scopeQuery(scope).toString()}`);
}

export async function getTicket(scope: PluginScope, ticketId: string): Promise<PluginTicketDetail> {
    return request<PluginTicketDetail>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}?${scopeQuery(scope).toString()}`);
}

// getTicketLinkContext загружает заявку для страницы, открытой по deep-link
// /plug/issuetrack/ticket/<uuid>. Канал не передаётся: после перехода на маршрут
// плагина его нет в клиенте, поэтому сервер определяет реалм по заявке и сам
// возвращает channelId для остальных запросов.
export async function getTicketLinkContext(userId: string, ticketId: string): Promise<PluginTicketLinkContext> {
    const qs = new URLSearchParams({userId});
    return request<PluginTicketLinkContext>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/link-context?${qs.toString()}`);
}

export async function setTicketStatus(scope: PluginScope, ticketId: string, status: string): Promise<void> {
    await request(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/status`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({channelId: scope.channelId, userId: scope.userId, botUserId: scope.botUserId || '', status}),
    });
}

export async function getComments(scope: PluginScope, ticketId: string): Promise<PluginComment[]> {
    return request<PluginComment[]>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/comments?${scopeQuery(scope).toString()}`);
}

export async function postComment(scope: PluginScope, ticketId: string, text: string, files: File[]): Promise<PluginComment> {
    const fd = new FormData();
    fd.append('channelId', scope.channelId);
    fd.append('userId', scope.userId);
    if (scope.botUserId) {
        fd.append('botUserId', scope.botUserId);
    }
    fd.append('text', text);
    for (const file of files || []) {
        fd.append('files', file);
    }
    return request<PluginComment>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/comments`, {method: 'POST', body: fd});
}

export function attachmentUrl(scope: PluginScope, attachmentId: string): string {
    return `/plugins/issuetrack/api/attachments/${encodeURIComponent(attachmentId)}?${scopeQuery(scope).toString()}`;
}

export async function createTicket(scope: PluginScope, payload: CreateTicketPayload): Promise<PluginCreateResult> {
    const fd = new FormData();
    fd.append('channelId', scope.channelId);
    fd.append('userId', scope.userId);
    if (scope.botUserId) {
        fd.append('botUserId', scope.botUserId);
    }
    fd.append('title', payload.title);
    if (payload.description) {
        fd.append('description', payload.description);
    }
    if (payload.categoryId) {
        fd.append('categoryId', payload.categoryId);
    }
    if (payload.siteId) {
        fd.append('siteId', payload.siteId);
    }
    if (payload.priority) {
        fd.append('priority', payload.priority);
    }
    if (payload.groupId) {
        fd.append('groupId', payload.groupId);
    }
    if (payload.assigneeId) {
        fd.append('assigneeId', payload.assigneeId);
    }
    if (payload.ownerId) {
        fd.append('ownerId', payload.ownerId);
    }
    if (payload.dueDate) {
        fd.append('dueDate', payload.dueDate);
    }
    for (const file of payload.files || []) {
        fd.append('files', file);
    }
    return request<PluginCreateResult>(`/plugins/issuetrack/api/tickets?${scopeQuery(scope).toString()}`, {method: 'POST', body: fd});
}