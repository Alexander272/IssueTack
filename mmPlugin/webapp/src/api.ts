import type {
    PluginApiErrorBody,
    PluginComment,
    PluginContextResult,
    PluginCreateResult,
    PluginStore,
    PluginTicketDetail,
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
}

const contextCache = new Map<string, {userId: string; ts: number; data: PluginContextResult}>();
const CONTEXT_TTL_MS = 5 * 60 * 1000;

export function getCachedContext(channelId: string | null, userId: string | null): PluginContextResult | null {
    if (!channelId || !userId) {
        return null;
    }
    const cached = contextCache.get(channelId);
    if (cached && cached.userId === userId && Date.now() - cached.ts < CONTEXT_TTL_MS) {
        return cached.data;
    }
    return null;
}

export async function getContext(channelId: string, userId: string): Promise<PluginContextResult> {
    const cached = getCachedContext(channelId, userId);
    if (cached) {
        return cached;
    }
    const data = await request<PluginContextResult>('/plugins/issuetrack/api/context', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({channelId, userId}),
    });
    contextCache.set(channelId, {userId, ts: Date.now(), data});
    return data;
}

export async function getContextFresh(channelId: string, userId: string): Promise<PluginContextResult> {
    contextCache.delete(channelId);
    return getContext(channelId, userId);
}

export async function getMyTickets(channelId: string, userId: string): Promise<PluginTicketShort[]> {
    const qs = new URLSearchParams({channelId, userId});
    return request<PluginTicketShort[]>(`/plugins/issuetrack/api/tickets?${qs.toString()}`);
}

export async function getTicket(channelId: string, userId: string, ticketId: string): Promise<PluginTicketDetail> {
    const qs = new URLSearchParams({channelId, userId});
    return request<PluginTicketDetail>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}?${qs.toString()}`);
}

export async function setTicketStatus(channelId: string, userId: string, ticketId: string, status: string): Promise<void> {
    await request(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/status`, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({channelId, userId, status}),
    });
}

export async function getComments(channelId: string, userId: string, ticketId: string): Promise<PluginComment[]> {
    const qs = new URLSearchParams({channelId, userId});
    return request<PluginComment[]>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/comments?${qs.toString()}`);
}

export async function postComment(channelId: string, userId: string, ticketId: string, text: string, files: File[]): Promise<PluginComment> {
    const fd = new FormData();
    fd.append('channelId', channelId);
    fd.append('userId', userId);
    fd.append('text', text);
    for (const file of files || []) {
        fd.append('files', file);
    }
    return request<PluginComment>(`/plugins/issuetrack/api/tickets/${encodeURIComponent(ticketId)}/comments`, {method: 'POST', body: fd});
}

export function attachmentUrl(channelId: string, userId: string, attachmentId: string): string {
    const qs = new URLSearchParams({channelId, userId});
    return `/plugins/issuetrack/api/attachments/${encodeURIComponent(attachmentId)}?${qs.toString()}`;
}

export async function createTicket(channelId: string, userId: string, payload: CreateTicketPayload): Promise<PluginCreateResult> {
    const fd = new FormData();
    fd.append('channelId', channelId);
    fd.append('userId', userId);
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
    for (const file of payload.files || []) {
        fd.append('files', file);
    }
    return request<PluginCreateResult>('/plugins/issuetrack/api/tickets', {method: 'POST', body: fd});
}