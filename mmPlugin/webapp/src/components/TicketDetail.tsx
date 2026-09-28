import React, { useCallback, useEffect, useRef, useState } from 'react'

import { ApiError, attachmentUrl, getComments, getTicket, postComment, setTicketStatus } from '../api'
import type { PluginComment, PluginCommentAttachment, PluginTicketDetail, PluginUserShort } from '../types'
import { formatBytes, formatDate, statusMeta } from '../labels'
import { OpenExternalIcon, TrashIcon } from './icons'

const MAX_PLUGIN_FILES = 10

function userName(u?: PluginUserShort): string {
    if (!u) {
        return ''
    }
    const full = [u.firstName, u.lastName].filter(Boolean).join(' ').trim()
    return full || u.username || '—'
}

interface TicketDetailProps {
    channelId: string
    userId: string
    ticketId: string
    onBack: () => void
    onChanged: () => void
}

export default function TicketDetail({ channelId, userId, ticketId, onBack, onChanged }: TicketDetailProps) {
    const [detail, setDetail] = useState<PluginTicketDetail | null>(null)
    const [detailError, setDetailError] = useState<string | null>(null)

    const [comments, setComments] = useState<PluginComment[] | null>(null)
    const [commentsError, setCommentsError] = useState<string | null>(null)

    const [commentText, setCommentText] = useState('')
    const [commentFiles, setCommentFiles] = useState<File[]>([])
    const [sending, setSending] = useState(false)
    const [sendError, setSendError] = useState<string | null>(null)
    const fileInputRef = useRef<HTMLInputElement>(null)

    const [changingStatus, setChangingStatus] = useState(false)
    const [statusError, setStatusError] = useState<string | null>(null)
    const [reasonOpen, setReasonOpen] = useState(false)
    const [reasonText, setReasonText] = useState('')
    const [confirmingCancel, setConfirmingCancel] = useState(false)
    const detailReqRef = useRef(0)
    const commentReqRef = useRef(0)

    const loadComments = useCallback(() => {
        const reqId = ++commentReqRef.current
        setComments(null)
        setCommentsError(null)
        getComments(channelId, userId, ticketId)
            .then(list => {
                if (reqId === commentReqRef.current) {
                    setComments(list || [])
                }
            })
            .catch((err: unknown) => {
                if (reqId === commentReqRef.current) {
                    setCommentsError(err instanceof ApiError ? err.message : 'Не удалось загрузить комментарии')
                }
            })
    }, [channelId, userId, ticketId])

    useEffect(() => {
        const reqId = ++detailReqRef.current
        setDetail(null)
        setDetailError(null)
        getTicket(channelId, userId, ticketId)
            .then(d => {
                if (reqId === detailReqRef.current) {
                    setDetail(d)
                }
            })
            .catch((err: unknown) => {
                if (reqId === detailReqRef.current) {
                    setDetailError(err instanceof ApiError ? err.message : 'Не удалось загрузить заявку')
                }
            })
        loadComments()
    }, [channelId, userId, ticketId, loadComments])

    // Смена статуса из плагина (действия владельца). При «Вернуть в работу»
    // (resolved → in_progress) сначала меняем статус и только потом постим
    // комментарий-причину: на ещё не активной заявке внешние комментарии запрещены.
    const applyStatus = useCallback(
        async (status: string, reason?: string) => {
            setChangingStatus(true)
            setStatusError(null)
            try {
                await setTicketStatus(channelId, userId, ticketId, status)
            } catch (err) {
                setStatusError(err instanceof ApiError ? err.message : 'Не удалось изменить статус')
                setChangingStatus(false)
                return
            }
            if (reason) {
                try {
                    await postComment(channelId, userId, ticketId, reason, [])
                } catch (err) {
                    setStatusError(
                        err instanceof ApiError ? err.message : 'Статус изменён, но причину не удалось отправить',
                    )
                }
            }
            setReasonOpen(false)
            setReasonText('')
            getTicket(channelId, userId, ticketId)
                .then(d => setDetail(d))
                .catch(() => undefined)
            loadComments()
            onChanged()
            setChangingStatus(false)
        },
        [channelId, userId, ticketId, loadComments, onChanged],
    )

    const sendComment = useCallback(() => {
        if (!commentText.trim() && commentFiles.length === 0) {
            return
        }
        setSending(true)
        setSendError(null)
        postComment(channelId, userId, ticketId, commentText, commentFiles)
            .then(() => {
                setCommentText('')
                setCommentFiles([])
                if (fileInputRef.current) {
                    fileInputRef.current.value = ''
                }
                loadComments()
            })
            .catch((err: unknown) =>
                setSendError(err instanceof ApiError ? err.message : 'Не удалось отправить комментарий'),
            )
            .finally(() => setSending(false))
    }, [channelId, userId, ticketId, commentText, commentFiles, loadComments])

    const onPickFiles = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const picked = Array.from(e.target.files || [])
        setCommentFiles(prev => prev.concat(picked).slice(0, MAX_PLUGIN_FILES))
        e.target.value = ''
    }, [])

    const removeFile = useCallback((index: number) => {
        setCommentFiles(prev => prev.filter((_, i) => i !== index))
    }, [])

    const detailStatus = detail ? statusMeta(detail.status) : null

    return (
        <div className='it-ticket-detail'>
            <div className='it-ticket-detail__bar'>
                <button type='button' className='it-btn it-btn--sm' onClick={onBack}>
                    ← Назад к заявкам
                </button>
                <div className='it-ticket-detail__bar-actions'>
                    {detail && detail.link ? (
                        <a
                            className='it-icon-btn'
                            href={detail.link}
                            target='_blank'
                            rel='noreferrer'
                            title='Открыть в программе'
                        >
                            <OpenExternalIcon size={18} />
                        </a>
                    ) : null}
                    {detail && detail.canCancel ? (
                        <button
                            type='button'
                            className='it-icon-btn it-icon-btn--danger'
                            title='Отменить заявку'
                            disabled={changingStatus}
                            onClick={() => {
                                setStatusError(null)
                                setConfirmingCancel(true)
                            }}
                        >
                            <TrashIcon size={16} />
                        </button>
                    ) : null}
                </div>
            </div>

            {detailError ? <div className='it-ticket-error'>{detailError}</div> : null}

            {!detail && !detailError ? <div className='it-ticket-detail__loading'>Загрузка…</div> : null}

            {detail && detailStatus ? (
                <div className='it-ticket-detail__body'>
                    <div className='it-ticket-detail__head'>
                        {detail.number ? <span className='it-ticket-detail__num'>№{detail.number}</span> : null}
                        <span className='it-ticket-detail__title'>{detail.title}</span>
                    </div>
                    <div className='it-ticket-detail__badges'>
                        <span
                            className='it-badge it-badge--status'
                            style={{ background: detailStatus.bg, color: detailStatus.text }}
                        >
                            {detailStatus.label}
                        </span>
                    </div>
                    {detail.canConfirm || detail.canReopen ? (
                        <div className='it-ticket-detail__actions'>
                            {detail.canConfirm ? (
                                <button
                                    type='button'
                                    className='it-btn it-btn--sm it-btn--primary'
                                    disabled={changingStatus}
                                    onClick={() => applyStatus('closed')}
                                >
                                    Подтвердить решение
                                </button>
                            ) : null}
                            {detail.canReopen ? (
                                <button
                                    type='button'
                                    className='it-btn it-btn--sm'
                                    disabled={changingStatus}
                                    onClick={() => {
                                        setStatusError(null)
                                        setReasonOpen(v => !v)
                                    }}
                                >
                                    {reasonOpen ? 'Отмена' : 'Вернуть в работу'}
                                </button>
                            ) : null}
                        </div>
                    ) : null}
                    {statusError ? <div className='it-ticket-error'>{statusError}</div> : null}
                    {detail.canReopen && reasonOpen ? (
                        <div className='it-reason'>
                            <div className='it-reason__title'>Причина возврата</div>
                            <textarea
                                className='it-comment-form__text it-reason__text'
                                placeholder='Зачем вернуть в работу…'
                                value={reasonText}
                                onChange={e => setReasonText(e.target.value)}
                            />
                            <button
                                type='button'
                                className='it-btn it-btn--sm it-btn--primary'
                                disabled={changingStatus || !reasonText.trim()}
                                onClick={() => applyStatus('in_progress', reasonText.trim())}
                            >
                                {changingStatus ? 'Отправка…' : 'Вернуть в работу'}
                            </button>
                        </div>
                    ) : null}
                    {detail.description ? <div className='it-ticket-detail__desc'>{detail.description}</div> : null}
                    <div className='it-ticket-detail__meta'>
                        {userName(detail.creator) ? (
                            <div className='it-ticket-detail__row'>
                                <span className='it-ticket-detail__label'>Заказчик</span>
                                <span className='it-ticket-detail__value'>{userName(detail.creator)}</span>
                            </div>
                        ) : null}
                        {detail.site && detail.site.name ? (
                            <div className='it-ticket-detail__row'>
                                <span className='it-ticket-detail__label'>Площадка</span>
                                <span className='it-ticket-detail__value'>{detail.site.name}</span>
                            </div>
                        ) : null}
                        {userName(detail.assignee) ? (
                            <div className='it-ticket-detail__row'>
                                <span className='it-ticket-detail__label'>Исполнитель</span>
                                <span className='it-ticket-detail__value'>{userName(detail.assignee)}</span>
                            </div>
                        ) : null}
                        <div className='it-ticket-detail__row'>
                            <span className='it-ticket-detail__label'>Создана</span>
                            <span className='it-ticket-detail__value'>{formatDate(detail.createdAt)}</span>
                        </div>
                    </div>
                </div>
            ) : null}

            {detail ? (
                <div className='it-ticket-detail__section'>
                    <div className='it-ticket-detail__section-title'>Вложения</div>
                    {detail.attachments && detail.attachments.length > 0 ? (
                        <ul className='it-attachments'>
                            {detail.attachments.map(att => (
                                <li key={att.id} className='it-attachment'>
                                    <a
                                        className='it-attachment__link'
                                        href={attachmentUrl(channelId, userId, att.id)}
                                        target='_blank'
                                        rel='noreferrer'
                                    >
                                        <span className='it-attachment__name'>{att.fileName}</span>
                                        <span className='it-attachment__size'>{formatBytes(att.fileSize)}</span>
                                    </a>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <div className='it-ticket-detail__empty'>Вложений нет</div>
                    )}
                </div>
            ) : null}

            <div className='it-ticket-detail__section'>
                <div className='it-ticket-detail__section-title'>Комментарии</div>
                {commentsError ? <div className='it-ticket-error'>{commentsError}</div> : null}
                {comments && comments.length === 0 ? (
                    <div className='it-ticket-detail__empty'>Комментариев пока нет</div>
                ) : null}
                {comments ? (
                    <div className='it-comments'>
                        {comments.map(c => (
                            <div key={c.id} className='it-comment'>
                                <div className='it-comment__head'>
                                    <span className='it-comment__author'>{c.user ? c.user.name : '—'}</span>
                                    <span className='it-comment__date'>{formatDate(c.createdAt)}</span>
                                </div>
                                {c.text ? <div className='it-comment__text'>{c.text}</div> : null}
                                {c.attachments && c.attachments.length > 0 ? (
                                    <div className='it-comment__files'>
                                        {c.attachments.map((att: PluginCommentAttachment) => (
                                            <a
                                                key={att.id}
                                                className='it-attachment__link it-attachment__link--inline'
                                                href={attachmentUrl(channelId, userId, att.id)}
                                                target='_blank'
                                                rel='noreferrer'
                                            >
                                                <span className='it-attachment__name'>{att.fileName}</span>
                                                <span className='it-attachment__size'>
                                                    {formatBytes(att.fileSize)}
                                                </span>
                                            </a>
                                        ))}
                                    </div>
                                ) : null}
                            </div>
                        ))}
                    </div>
                ) : null}
            </div>

            <div className='it-comment-form'>
                {sendError ? <div className='it-ticket-error'>{sendError}</div> : null}
                <textarea
                    className='it-comment-form__text'
                    placeholder='Написать комментарий…'
                    value={commentText}
                    onChange={e => setCommentText(e.target.value)}
                />
                {commentFiles.length > 0 ? (
                    <ul className='it-files-picked'>
                        {commentFiles.map((f, i) => (
                            <li key={`${f.name}-${i}`} className='it-files-picked__item'>
                                <span className='it-files-picked__name'>{f.name}</span>
                                <button
                                    type='button'
                                    className='it-files-picked__remove'
                                    onClick={() => removeFile(i)}
                                    title='Удалить'
                                >
                                    ×
                                </button>
                            </li>
                        ))}
                    </ul>
                ) : null}
                <div className='it-comment-form__actions'>
                    <input
                        ref={fileInputRef}
                        type='file'
                        multiple
                        className='it-comment-form__file-input'
                        onChange={onPickFiles}
                    />
                    <button type='button' className='it-btn it-btn--sm' onClick={() => fileInputRef.current?.click()}>
                        Прикрепить файл
                    </button>
                    <button
                        type='button'
                        className='it-btn it-btn--primary it-btn--sm'
                        disabled={sending || (!commentText.trim() && commentFiles.length === 0)}
                        onClick={sendComment}
                    >
                        {sending ? 'Отправка…' : 'Отправить'}
                    </button>
                </div>
                <div className='it-comment-form__hint'>До {MAX_PLUGIN_FILES} файлов</div>
            </div>

            {confirmingCancel && detail ? (
                <div className='it-confirm' onClick={() => setConfirmingCancel(false)}>
                    <div className='it-confirm__box' onClick={e => e.stopPropagation()}>
                        <div className='it-confirm__title'>Отменить заявку?</div>
                        <div className='it-confirm__text'>
                            {detail.number ? `№${detail.number}. ` : ''}
                            {detail.title}. Действие необратимо.
                        </div>
                        <div className='it-confirm__actions'>
                            <button
                                type='button'
                                className='it-btn it-btn--sm'
                                disabled={changingStatus}
                                onClick={() => setConfirmingCancel(false)}
                            >
                                Нет
                            </button>
                            <button
                                type='button'
                                className='it-btn it-btn--sm it-btn--danger it-btn--solid'
                                disabled={changingStatus}
                                onClick={() => {
                                    setConfirmingCancel(false)
                                    applyStatus('cancelled')
                                }}
                            >
                                {changingStatus ? 'Отмена…' : 'Отменить'}
                            </button>
                        </div>
                    </div>
                </div>
            ) : null}
        </div>
    )
}