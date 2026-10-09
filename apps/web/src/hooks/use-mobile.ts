import { useSyncExternalStore } from 'react';

const MOBILE_BREAKPOINT = 768;
const NARROW = `(max-width: ${MOBILE_BREAKPOINT - 1}px)`;

function subscribe(onChange: () => void): () => void {
    const query = globalThis.matchMedia(NARROW);
    query.addEventListener('change', onChange);
    return () => {
        query.removeEventListener('change', onChange);
    };
}

function isNarrow(): boolean {
    return globalThis.innerWidth < MOBILE_BREAKPOINT;
}

function onTheServer(): boolean {
    return false;
}

export function useIsMobile(): boolean {
    return useSyncExternalStore(subscribe, isNarrow, onTheServer);
}
