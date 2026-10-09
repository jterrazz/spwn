import { useSyncExternalStore } from 'react';

function subscribeToSeconds(onTick: () => void): () => void {
    const id = setInterval(onTick, 1000);
    return () => {
        clearInterval(id);
    };
}

function currentSecond(): number {
    return Math.floor(Date.now() / 1000);
}

function serverSecond(): number {
    return 0;
}

/** The current time in whole seconds, re-rendering the component as each second passes. */
export function useClock(): number {
    return useSyncExternalStore(subscribeToSeconds, currentSecond, serverSecond);
}
