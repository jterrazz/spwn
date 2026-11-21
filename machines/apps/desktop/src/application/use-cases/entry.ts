import type { Target } from '@jterrazz/station-api';

import { EngineError } from '../../domain/engine-error';
import type { In, ReadName, Watch, WatchName } from '../ports/engine.port';
import { replaceEqualDeep } from './equal';
import type { Mark } from './marks';

export type Reading<T> =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: T; done: boolean }
    | { kind: 'reading' };

export const READING = { kind: 'reading' } as const;

export type Asked =
    | { kind: 'read'; name: ReadName; input: In<ReadName>; target: Target | undefined }
    | { kind: 'watch'; name: WatchName; input: In<WatchName>; target: Target | undefined };

// value: `asking` while a request is out and has not settled; `held`, a quiet re-ask's last out not shown yet.
export type Entry = {
    key: string;
    asked: Asked;
    mark: Mark | undefined;
    subscribe: (listener: () => void) => () => void;
    reading: Reading<unknown>;
    listeners: Set<() => void>;
    asks: number;
    asking: boolean;
    held: undefined | { out: unknown };
    watch: undefined | Watch;
    grace: ReturnType<typeof setTimeout> | undefined;
};

function sharedOf(before: Reading<unknown>, next: Reading<unknown>): Reading<unknown> {
    if (before.kind !== 'read' || next.kind !== 'read') {
        return next;
    }
    const out = replaceEqualDeep(before.out, next.out);
    return out === before.out && next.done === before.done ? before : { ...next, out };
}

export function show(entry: Entry, next: Reading<unknown>): void {
    const reading = sharedOf(entry.reading, next);
    if (reading === entry.reading) {
        return;
    }
    entry.reading = reading;
    for (const listener of entry.listeners) {
        listener();
    }
}

export function failed(entry: Entry, error: EngineError): void {
    if (error.wire || entry.reading.kind !== 'read') {
        show(entry, { error, kind: 'failed' });
    }
}

export function stop(entry: Entry): void {
    entry.asks += 1;
    entry.asking = false;
    entry.held = undefined;
    entry.watch?.close();
    entry.watch = undefined;
}

export function flush(entry: Entry): void {
    if (entry.held) {
        show(entry, { done: false, kind: 'read', out: entry.held.out });
        entry.held = undefined;
    }
}

export function failedOf(error: unknown): EngineError {
    return error instanceof EngineError ? error : new EngineError(undefined, String(error));
}
