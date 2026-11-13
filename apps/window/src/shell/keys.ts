import { useEffect } from 'react';
import type { KeyboardEvent } from 'react';

import { focusablesIn, moveFocus, moveOfKey } from '../design/focus';
import { aDialogIsOpen } from '../design/index';
import { machineRoute } from './route';
import type { Router } from './router';
import { shortcutOf } from './shortcut';
import type { Shortcut } from './shortcut';

function obey(shortcut: Shortcut, router: Router, onFind: () => void): void {
    const { route, go, back, forward } = router;
    switch (shortcut) {
        case 'back': {
            back();
            return;
        }
        case 'forward': {
            forward();
            return;
        }
        case 'home': {
            go({ page: 'home' });
            return;
        }
        case 'find': {
            onFind();
            return;
        }
        case 'gallery': {
            go({ page: 'gallery' });
            return;
        }
        default: {
            if (route.page === 'machine') {
                go(machineRoute(route.machine, shortcut));
            }
        }
    }
}

// invariant: while a dialog is open the window's shortcuts are swallowed, so a draft in it is never unmounted.
export function useShortcuts(router: Router, onFind: () => void, withGallery: boolean): void {
    useEffect(() => {
        const onKey = (event: globalThis.KeyboardEvent) => {
            const shortcut = shortcutOf(event, withGallery);
            if (shortcut === undefined) {
                return;
            }
            event.preventDefault();
            if (!aDialogIsOpen()) {
                obey(shortcut, router, onFind);
            }
        };
        globalThis.addEventListener('keydown', onKey);
        return () => {
            globalThis.removeEventListener('keydown', onKey);
        };
    }, [router, onFind, withGallery]);
}

export function onTabKey(event: KeyboardEvent<HTMLElement>): void {
    const tab = event.target;
    const move = moveOfKey(event.key, 'horizontal');
    if (!(tab instanceof HTMLElement) || tab.getAttribute('role') !== 'tab' || !move) {
        return;
    }
    const list = tab.closest('[role="tablist"]');
    if (!list) {
        return;
    }
    event.preventDefault();
    moveFocus(focusablesIn(list, '[role="tab"]:not(:disabled)'), move, true)?.click();
}

export function onListKey(event: KeyboardEvent<HTMLElement>): void {
    const move = moveOfKey(event.key, 'vertical');
    if (move) {
        event.preventDefault();
        moveFocus(focusablesIn(event.currentTarget, 'button:not(:disabled)'), move, false);
    }
}
