// postSeed.ts — предзаполнение формы создания заявки из сообщения Mattermost.
// registerPostActionComponent отдаёт компоненту готовый пост, поэтому читать
// его из стора не нужно. Наружу отдаём сериализуемое «семе»: текст, id автора
// и ссылки на файлы. Сами File скачиваются уже внутри формы — с сессией
// пользователя MM и с его правами на файлы канала.

import type {CreateTicketSeed, PluginPostFileRef, PluginPostStub} from './types';

// Ограничение textarea «Подробное описание» в форме плагина.
const MAX_DESCRIPTION_CHARS = 5000;

function truncate(text: string, max: number): string {
    const chars = Array.from(text);
    return chars.length <= max ? text : chars.slice(0, max).join('');
}

// canCreateFromPost отсекает системные и удалённые посты: у них нет полезного
// текста, а иконка на них только мешала бы. Пост только с файлами — валиден,
// вложения переносятся даже без текста.
export function canCreateFromPost(post: PluginPostStub | null): boolean {
    if (!post || post.delete_at || post.type) {
        return false;
    }
    return Boolean((post.message || '').trim() || (post.file_ids || []).length || (post.metadata?.files || []).length);
}

export function seedFromPost(post: PluginPostStub): CreateTicketSeed | null {
    if (!canCreateFromPost(post)) {
        return null;
    }
    const seed: CreateTicketSeed = {};
    const message = (post.message || '').trim();
    if (message) {
        seed.description = truncate(message, MAX_DESCRIPTION_CHARS);
    }
    if (post.user_id) {
        seed.authorMmId = post.user_id;
    }

    const infos = post.metadata?.files || [];
    const byId = new Map(infos.map(info => [info.id, info]));
    const ids = (post.file_ids || []).length ? post.file_ids! : infos.map(info => info.id);
    const files: PluginPostFileRef[] = [];
    const seen = new Set<string>();
    for (const id of ids || []) {
        if (!id || seen.has(id)) {
            continue;
        }
        seen.add(id);
        const info = byId.get(id);
        files.push({
            id,
            name: info?.name || id,
            ...(info?.mime_type ? {mimeType: info.mime_type} : {}),
            ...(typeof info?.size === 'number' ? {size: info.size} : {}),
        });
    }
    if (files.length) {
        seed.files = files;
    }
    return seed;
}
