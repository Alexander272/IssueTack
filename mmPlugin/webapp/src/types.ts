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

export interface PluginContextResult {
    bound: boolean;
    realmId: string;
    realmName: string;
    user: PluginUser;
    categories: PluginCategory[];
    sites: PluginSite[];
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
    attachments?: PluginAttachment[];
    canConfirm?: boolean;
    canReopen?: boolean;
    canCancel?: boolean;
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
        };
    };
}

export interface PluginRegistry {
    registerChannelHeaderIcon(input: {component: ComponentType<{channel: {id: string}}>}): string;
}

export interface IssuetrackWebappPlugin {
    initialize(registry: PluginRegistry, store: PluginStore): void;
}