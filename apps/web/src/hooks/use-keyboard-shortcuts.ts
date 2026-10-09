'use client';

import { useEffect } from 'react';

interface ShortcutHandlers {
    onSearch?: () => void;
    onSpawnWorld?: () => void;
    onEscape?: () => void;
}

const TYPING_TAGS = new Set(['INPUT', 'SELECT', 'TEXTAREA']);

function isTyping(event: KeyboardEvent): boolean {
    return event.target instanceof HTMLElement && TYPING_TAGS.has(event.target.tagName);
}

function handlerOf(event: KeyboardEvent, handlers: ShortcutHandlers): (() => void) | undefined {
    const command = event.metaKey || event.ctrlKey;
    if (command && event.key === 'k') {
        return handlers.onSearch;
    }
    if (command && event.key === 'n' && !isTyping(event)) {
        return handlers.onSpawnWorld;
    }
    return undefined;
}

/** Global keyboard shortcuts: ⌘K / Ctrl+K searches, ⌘N / Ctrl+N spawns a world outside a field, Escape closes. */
export function useKeyboardShortcuts(handlers: ShortcutHandlers) {
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                handlers.onEscape?.();
                return;
            }
            const handler = handlerOf(event, handlers);
            if (handler !== undefined) {
                event.preventDefault();
                handler();
            }
        };

        globalThis.addEventListener('keydown', handleKeyDown);
        return () => {
            globalThis.removeEventListener('keydown', handleKeyDown);
        };
    }, [handlers]);
}
