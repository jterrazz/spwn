import { useSyncExternalStore } from 'react';

function subscribeToNothing(): () => void {
    return () => undefined;
}

function onTheClient(): boolean {
    return true;
}

function onTheServer(): boolean {
    return false;
}

/** False while the server renders and the client hydrates, true once the page is the client's: the moment a browser-only value may be shown. */
export function useHydrated(): boolean {
    return useSyncExternalStore(subscribeToNothing, onTheClient, onTheServer);
}
