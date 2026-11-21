import type { Target } from '@jterrazz/station-api';

import { EngineError } from '../../domain/engine-error';
import type { In, ReadName, Watch, WatchName } from '../ports/engine.port';
import { replaceEqualDeep } from './equal';
import type { Mark } from './marks';

// value: `stale`, the error a re-ask met that could not read the answer again, the answer kept: a page draws it as a mark.
// value: `placeholder`, the answer of the input a hook asked before, shown while the new one is read.
export type Reading<T> =
    | { kind: 'failed'; error: EngineError }
    | { kind: 'read'; out: T; done: boolean; stale?: EngineError; placeholder?: true }
    | { kind: 'reading' };

export const READING = { kind: 'reading' } as const;

export type Asked =
    | { kind: 'read'; name: ReadName; input: In<ReadName>; target: Target | undefined }
    | { kind: 'watch'; name: WatchName; input: In<WatchName>; target: Target | undefined };

// value: `asking` while a request is out and has not settled; `held`, a quiet re-ask's last out not shown yet; `kept`,
// value: past its grace with nobody holding it, its request stopped and its answer kept.
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
    kept: boolean;
    watch: undefined | Watch;
    grace: ReturnType<typeof setTimeout> | undefined;
};

function sameError(before: EngineError | undefined, next: EngineError | undefined): boolean {
    if (before === undefined || next === undefined) {
        return before === next;
    }
    return (
        before.message === next.message &&
        before.fate === next.fate &&
        replaceEqualDeep(before.wire, next.wire) === before.wire
    );
}

function sharedOf(before: Reading<unknown>, next: Reading<unknown>): Reading<unknown> {
    if (before.kind === 'failed' && next.kind === 'failed') {
        return sameError(before.error, next.error) ? before : next;
    }
    if (before.kind !== 'read' || next.kind !== 'read') {
        return next;
    }
    const out = replaceEqualDeep(before.out, next.out);
    const same =
        out === before.out && next.done === before.done && sameError(before.stale, next.stale);
    return same ? before : { ...next, out };
}

// value: the refusals that say the answer could not be read again, not what the engine answers of its subject: no
// value: engine word (the request reached none, or its answer was lost), a far Mac that missed the ask, a call cut short.
function unread(error: EngineError): boolean {
    const type = error.wire?.type;
    return error.wire === undefined || type === 'unreachable' || type === 'interrupted';
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
    const { reading } = entry;
    if (reading.kind === 'read' && unread(error)) {
        show(entry, { ...reading, stale: error });
    } else {
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
