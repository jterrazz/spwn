import type { EngineSliceOut, EngineStateFrame, EngineStateSince } from '@jterrazz/station-api';

import { EngineError } from '../../domain/engine-error';
import { STATE_REVISION } from '../../domain/fit';
import { isOfMachine, isView, VIEW_KEYS, viewNamed } from '../../domain/views';
import type { View } from '../../domain/views';
import type { EnginePort, Link, Watch } from '../ports/engine.port';
import { READING } from './entry';
import type { Asked, Reading } from './entry';
import { replaceEqualDeep } from './equal';
import { judged } from './reach';

// value: `lost`, how long a held stream may send no frame before it is asked again from the revision held: three of
// value: the engine's 15 s ticks; `retry`, how long after a cut connection it is asked again; `grace`, how long the
// value: stream stays open once nobody holds a view.
export type MirrorTiming = { lost: number; retry: number; grace: number };

const DEFAULTS: MirrorTiming = { grace: 1500, lost: 45_000, retry: 1000 };

// value: the why of a machine's view the engine's stream does not hold: the engine reads no such machine, so its call
// value: answers.
const NOT_KEPT = 'not-kept';

// value: the view a read is, by the engine's naming (`domain/views.ts`); undefined is a read asked by its call.
export function viewOf(asked: Asked): undefined | View {
    return viewNamed(asked.name, asked.input, asked.target);
}

function staleOf(slice: EngineSliceOut): EngineError | undefined {
    return slice.state === 'stale' ? new EngineError(undefined, slice.why ?? 'stale') : undefined;
}

function atOf(slice: EngineSliceOut, held: Reading<unknown> | undefined): Date {
    if (slice.read_at !== undefined) {
        return new Date(slice.read_at);
    }
    return held?.kind === 'read' ? held.at : new Date(0);
}

// value: the why of a view the engine could not read and holds no reading of: its call answers the refusal, so the
// value: window asks it; `reading` is a view still being read, which draws its skeleton.
function refusedOf(slice: EngineSliceOut, held: Reading<unknown>): string | undefined {
    const refused =
        slice.state === 'unread' &&
        slice.value === undefined &&
        slice.why !== undefined &&
        slice.why !== 'reading' &&
        held.kind !== 'read';
    return refused ? slice.why : undefined;
}

function sameMark(held: Reading<unknown>, stale: EngineError | undefined): boolean {
    return held.kind === 'read' && held.stale?.message === stale?.message;
}

// invariant: a slice replaces the reading it moves and keeps the parts that did not: an equal value is the one shown,
// invariant: a slice sent without its value keeps the value held, and a slice not read yet draws nothing new.
function readingOf(before: Reading<unknown>, slice: EngineSliceOut): Reading<unknown> {
    const held = before.kind === 'read' ? before : undefined;
    if (slice.value === undefined && held === undefined) {
        return before;
    }
    const out = replaceEqualDeep(held?.out, slice.value ?? held?.out);
    const stale = staleOf(slice);
    if (held && out === held.out && sameMark(held, stale)) {
        return held;
    }
    const at = atOf(slice, held);
    return stale
        ? { at, done: true, kind: 'read', out, stale }
        : { at, done: true, kind: 'read', out };
}

// invariant: ADR-071's mirror: one held engine.state stream of the views, opened while a view is held and the link
// invariant: is open and fits, its readings kept across a lost link and a restart, so a page once drawn never blanks;
// invariant: a stream silent past `lost`, or cut, is asked again from the revision held, and answers what moved.
export class StateMirror {
    private readonly client: EnginePort;
    private readonly timing: MirrorTiming;
    private readonly readings = new Map<View, Reading<unknown>>();
    private readonly refused = new Map<View, string>();
    private readonly listeners = new Map<View, Set<() => void>>();
    private readonly subscribers = new Map<View, (listener: () => void) => () => void>();
    private open = false;
    private held = 0;
    private since: EngineStateSince | undefined;
    private kept: Set<View> | undefined;
    private watch: undefined | Watch;
    private asking: object | undefined;
    private silent: ReturnType<typeof setTimeout> | undefined;
    private again: ReturnType<typeof setTimeout> | undefined;
    private grace: ReturnType<typeof setTimeout> | undefined;
    private unlink: (() => void) | undefined;

    constructor(client: EnginePort, timing: Partial<MirrorTiming> = {}) {
        this.client = client;
        this.timing = { ...DEFAULTS, ...timing };
    }

    readonly reading = (view: View): Reading<unknown> => this.readings.get(view) ?? READING;

    readonly refusedOf = (view: View): string | undefined => this.refused.get(view);

    // value: the same function for a view every time, so React keeps its subscription.
    readonly subscriberOf = (view: View): ((listener: () => void) => () => void) => {
        let subscriber = this.subscribers.get(view);
        if (!subscriber) {
            subscriber = (listener) => this.subscribe(view, listener);
            this.subscribers.set(view, subscriber);
        }
        return subscriber;
    };

    private subscribe(view: View, listener: () => void): () => void {
        const listeners = this.listeners.get(view) ?? new Set();
        this.listeners.set(view, listeners);
        listeners.add(listener);
        this.held += 1;
        this.absent(view);
        clearTimeout(this.grace);
        this.unlink ??= this.client.link((link) => {
            this.heard(link);
        });
        this.follow();
        return () => {
            listeners.delete(listener);
            this.held -= 1;
            if (this.held === 0) {
                this.grace = setTimeout(() => {
                    this.release();
                }, this.timing.grace);
            }
        };
    }

    private release(): void {
        if (this.held > 0) {
            return;
        }
        this.close();
        this.unlink?.();
        this.unlink = undefined;
        this.open = false;
    }

    private heard(link: Link): void {
        this.open = judged(link).kind === 'open';
        if (!this.open) {
            this.close();
        }
        this.follow();
    }

    private follow(): void {
        if (this.open && this.held > 0 && this.watch === undefined) {
            this.ask();
        }
    }

    private close(): void {
        clearTimeout(this.silent);
        clearTimeout(this.again);
        this.watch?.close();
        this.watch = undefined;
        this.asking = undefined;
    }

    private ask(): void {
        const { since } = this;
        const input = since
            ? { hold: true, keys: [...VIEW_KEYS], revision: STATE_REVISION, since }
            : { hold: true, keys: [...VIEW_KEYS], revision: STATE_REVISION };
        const mine = {};
        this.asking = mine;
        const current = () => this.asking === mine;
        const watch = this.client.watch('engine.state', input, {
            done: () => {
                if (current()) {
                    this.close();
                }
            },
            error: (error) => {
                if (!current()) {
                    return;
                }
                this.close();
                if (error.wire === undefined) {
                    this.again = setTimeout(() => {
                        this.follow();
                    }, this.timing.retry);
                    return;
                }
                for (const view of this.listeners.keys()) {
                    this.show(view, { error, kind: 'failed' });
                }
            },
            out: (frame) => {
                if (current()) {
                    this.frame(frame);
                }
            },
        });
        if (!current()) {
            watch.close();
            return;
        }
        this.watch = watch;
        this.alive();
    }

    // invariant: every frame is the stream's liveness; one missing past `lost` is a lost stream, asked again.
    private alive(): void {
        clearTimeout(this.silent);
        this.silent = setTimeout(() => {
            this.close();
            this.follow();
        }, this.timing.lost);
    }

    private frame(frame: EngineStateFrame): void {
        this.alive();
        if (frame.event === 'opened') {
            this.opened(frame);
            return;
        }
        const kept = this.kept ?? new Set<View>();
        this.kept = kept;
        if (this.since) {
            this.since = { ...this.since, rev: frame.rev };
        }
        for (const [key, slice] of Object.entries(frame.slices ?? {})) {
            if (isView(key)) {
                kept.add(key);
                this.refuse(key, refusedOf(slice, this.reading(key)));
                this.show(key, readingOf(this.reading(key), slice));
            }
        }
        for (const key of frame.gone ?? []) {
            if (isView(key)) {
                kept.delete(key);
            }
        }
        for (const view of this.listeners.keys()) {
            this.absent(view);
        }
    }

    private opened(frame: EngineStateFrame): void {
        if (frame.fit?.verdict !== 'fits') {
            this.close();
        }
        this.since = frame.epoch === undefined ? undefined : { epoch: frame.epoch, rev: frame.rev };
        this.kept = undefined;
    }

    // invariant: from the first frame of a stream on, a machine's view it does not hold is refused to its call.
    private absent(view: View): void {
        if (this.kept !== undefined && isOfMachine(view) && !this.kept.has(view)) {
            this.refuse(view, NOT_KEPT);
        }
    }

    private refuse(view: View, why: string | undefined): void {
        if (this.refused.get(view) === why) {
            return;
        }
        if (why === undefined) {
            this.refused.delete(view);
        } else {
            this.refused.set(view, why);
        }
        for (const listener of this.listeners.get(view) ?? []) {
            listener();
        }
    }

    private show(view: View, next: Reading<unknown>): void {
        if (next === this.readings.get(view) || (next === READING && !this.readings.has(view))) {
            return;
        }
        this.readings.set(view, next);
        for (const listener of this.listeners.get(view) ?? []) {
            listener();
        }
    }
}
