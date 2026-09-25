import type {Priority, TicketStatus} from './types';

interface BadgeStyle {
    label: string;
    bg: string;
    text: string;
}

export const STATUS_MAP: Record<TicketStatus, BadgeStyle> = {
    open: {label: 'Новая', bg: '#E1F5FE', text: '#01579B'},
    in_progress: {label: 'В работе', bg: '#FFF3E0', text: '#E65100'},
    pending: {label: 'Ожидание', bg: '#FFFDE7', text: '#F57F17'},
    on_hold: {label: 'Отложена', bg: '#F3E5F5', text: '#4A148C'},
    resolved: {label: 'Решена', bg: '#E8F5E9', text: '#1B5E20'},
    closed: {label: 'Закрыта', bg: '#C8E6C9', text: '#024a02'},
    cancelled: {label: 'Отменена', bg: '#FFEBEE', text: '#B71C1C'},
};

export const PRIORITY_MAP: Record<Priority, BadgeStyle> = {
    low: {label: 'Низкий', bg: '#ECFDF5', text: '#065F46'},
    medium: {label: 'Средний', bg: '#FEF3C7', text: '#92400E'},
    high: {label: 'Высокий', bg: '#FEE2E2', text: '#B91C1C'},
    urgent: {label: 'Критичный', bg: '#FEF2F2', text: '#991B1B'},
};

const FALLBACK_BADGE: BadgeStyle = {label: '', bg: '#EEEEEE', text: '#424242'};

export function statusMeta(status: TicketStatus): BadgeStyle {
    return STATUS_MAP[status] || {...FALLBACK_BADGE, label: status || ''};
}

export function priorityMeta(priority: Priority): BadgeStyle {
    return PRIORITY_MAP[priority] || {...FALLBACK_BADGE, label: priority || ''};
}

export function formatDate(iso: string): string {
    if (!iso) {
        return '';
    }
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
        return '';
    }
    return d.toLocaleDateString('ru-RU');
}

export function formatBytes(bytes: number): string {
    if (bytes === null || bytes === undefined) {
        return '';
    }
    const units = ['Б', 'КБ', 'МБ', 'ГБ'];
    let value = bytes;
    let i = 0;
    while (value >= 1024 && i < units.length - 1) {
        value /= 1024;
        i += 1;
    }
    return `${value.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}