// CreateTicketPostAction — иконка «Создать заявку» на панели действий сообщения
// (registerPostActionComponent). MM рендерит её при наведении (или при
// клавиатурной навигации) рядом с реакциями и ответом, поэтому действие в один
// клик доступно без открытия меню сообщения.

import React from 'react';

import appIcon from '../assets/icon.jpg';
import {requestCreateWithSeed} from '../deepLink';
import {seedFromPost} from '../postSeed';
import type {PluginPostStub} from '../types';

interface CreateTicketPostActionProps {
    post: PluginPostStub;
    handleDropdownOpened?: (open: boolean) => void;
}

export default function CreateTicketPostAction({post, handleDropdownOpened}: CreateTicketPostActionProps) {
    const seed = seedFromPost(post);
    if (!seed) {
        return null;
    }

    return (
        <button
            type='button'
            className='it-post-action post-menu__item'
            title='Создать заявку в службу IT'
            aria-label='Создать заявку в службу IT'
            data-testid='it-create-ticket-post-action'
            onClick={e => {
                // Клик по иконке не должен попадать в обработчики самого поста.
                e.preventDefault();
                e.stopPropagation();
                handleDropdownOpened?.(false);
                requestCreateWithSeed(seed);
            }}
        >
            {/* Та же бренд-иконка, что в App Bar; webpack инлайнит её в data-URI. */}
            <img className='it-post-action__icon' src={appIcon} alt='' />
        </button>
    );
}
