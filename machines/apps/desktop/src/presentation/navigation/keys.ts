import { useEffect } from 'react';
import type { KeyboardEvent } from 'react';

import { focusablesIn, moveFocus, moveOfKey } from '../ui/focus';
import { aDialogIsOpen } from '../ui/index';
import { machineRoute } from './route';
import type { Router } from './router';
import { mouseShortcutOf, shortcutOf } from './shortcut';
import type { Shortcut } from './shortcut';
import type { Space } from './spaces';

export type Acts = {
    onCard: () => void;
    onColumn: () => void;
    onFind: () => void;
    onSpace: (space: Space) => void;
};

function obey(shortcut: Shortcut, router: Router, acts: Acts): void {
    const { onCard, onColumn, onFind, onSpace } = acts;
    const { route, go, back, forward } = router;
    switch (shortcut) {
        case 'card': {
            onCard();
            return;
        }
        case 'column': {
            onColumn();
            return;
        }
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
        case 'settings': {
            go({ page: 'settings' });
            return;
        }
        case 'inbox': {
            go({ page: 'inbox' });
            return;
        }
        case 'space-machine': {
            onSpace('machine');
            return;
        }
        case 'space-agent': {
            onSpace('agent');
            return;
        }
        case 'space-os': {
            onSpace('os');
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

// invariant: while a dialog is open the window's shortcuts are swallowed, so a draft in it is never unmounted; while
// invariant: the frame holds the no-engine panel (`on` false) none of them acts.
export function useShortcuts(router: Router, acts: Acts, withGallery: boolean, on: boolean): void {
    useEffect(() => {
        const onKey = (event: globalThis.KeyboardEvent) => {
            const shortcut = shortcutOf(event, withGallery);
            if (shortcut === undefined || !on) {
                return;
            }
            event.preventDefault();
            if (!aDialogIsOpen()) {
                obey(shortcut, router, acts);
            }
        };
        const onMouse = (event: MouseEvent) => {
            const shortcut = mouseShortcutOf(event.button);
            if (shortcut === undefined || !on) {
                return;
            }
            event.preventDefault();
            if (!aDialogIsOpen()) {
                obey(shortcut, router, acts);
            }
        };
        globalThis.addEventListener('keydown', onKey);
        globalThis.addEventListener('mouseup', onMouse);
        return () => {
            globalThis.removeEventListener('keydown', onKey);
            globalThis.removeEventListener('mouseup', onMouse);
        };
    }, [router, acts, withGallery, on]);
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
