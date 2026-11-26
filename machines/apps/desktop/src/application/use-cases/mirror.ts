import type { EngineHello, EngineStateFrame, EngineStateSince } from '@jterrazz/station-api';

import { STATE_REVISION } from '../../domain/fit';
import type { Fit } from '../../domain/fit';
import { isOfMachine, isView, VIEW_KEYS } from '../../domain/views';
import type { View } from '../../domain/views';
import type { EnginePort, Link, Watch } from '../ports/engine.port';
import { READING } from './entry';
import type { Reading } from './entry';
import { sameEngine } from './fresh';
import { readingOf, refusedOf } from './sliced';

// value: `lost`, how long a held stream may send no frame before it is asked again from the revision held: three of
// value: the engine's 15 s ticks; `retry`, how long after a cut connection it is asked again; `grace`, how long the
// value: stream stays open once nobody holds a view.
export type MirrorTiming = { lost: number; retry: number; grace: number };

const DEFAULTS: MirrorTiming = { grace: 1500, lost: 45_000, retry: 1000 };

// value: the why of a machine's view the engine's stream does not hold: the engine reads no such machine, so its call
// value: answers.
const NOT_KEPT = 'not-kept';

// invariant: ADR-071's mirror: one held engine.state stream of the views, opened while a view or the verdict is held
// invariant: and the link is open, its readings kept across a lost link and a restart, so a page once drawn never
// invariant: blanks; a stream silent past `lost`, or cut, is asked again from the revision held, and answers what
// invariant: moved; one the engine refused is drawn as each view's failure in its words, and asked again past `lost`. Its `opened` judges the engine it reaches (ADR-070's one verdict, none named an older engine); a
// invariant: misfit closes it until another engine answers.
export class StateMirror {
    private readonly client: EnginePort;
    private readonly timing: MirrorTiming;
    private readonly readings = new Map<View, Reading<unknown>>();
    private readonly refused = new Map<View, string>();
    private readonly listeners = new Map<View, Set<() => void>>();
    private readonly subscribers = new Map<View, (listener: () => void) => () => void>();
    private readonly judges = new Set<() => void>();
    private readonly revisions = new Set<(views: readonly View[]) => void>();
    private verdict: Fit | undefined;
    private judgedOf: EngineHello | undefined;
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

    readonly fitOf = (hello: EngineHello | undefined): Fit | undefined =>
        sameEngine(this.judgedOf, hello) ? this.verdict : undefined;

    readonly keeps = (view: View): boolean => this.kept?.has(view) === true;

    readonly subscribeRevised = (listener: (views: readonly View[]) => void): (() => void) => {
        this.revisions.add(listener);
        return () => {
            this.revisions.delete(listener);
        };
    };

    readonly subscribeFit = (listener: () => void): (() => void) => {
        this.judges.add(listener);
        const leave = this.hold();
        return () => {
            this.judges.delete(listener);
            leave();
        };
    };

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
        const leave = this.hold();
        this.absent(view);
        return () => {
            listeners.delete(listener);
            leave();
        };
    }

    private hold(): () => void {
        this.held += 1;
        clearTimeout(this.grace);
        this.unlink ??= this.client.link((link) => {
            this.heard(link);
        });
        this.follow();
        return () => {
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
        this.open = link.kind === 'open';
        if (link.kind !== 'open') {
            this.close();
        } else if (!sameEngine(this.judgedOf, link.hello)) {
            this.close();
            this.judgedOf = link.hello;
            this.verdict = undefined;
        }
        this.follow();
    }

    private follow(): void {
        const unfit = this.verdict !== undefined && this.verdict !== 'fits';
        if (this.open && !unfit && this.held > 0 && this.watch === undefined) {
            this.ask();
        }
    }

    private judge(verdict: Fit): void {
        if (this.verdict === verdict) {
            return;
        }
        this.verdict = verdict;
        for (const listener of this.judges) {
            listener();
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
                const refused = error.wire !== undefined;
                if (refused) {
                    for (const view of this.listeners.keys()) {
                        this.show(view, { error, kind: 'failed' });
                    }
                }
                this.again = setTimeout(
                    () => {
                        this.follow();
                    },
                    refused ? this.timing.lost : this.timing.retry,
                );
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
        const revised: View[] = [];
        for (const [key, slice] of Object.entries(frame.slices ?? {})) {
            if (isView(key)) {
                revised.push(key);
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
        this.revise(revised);
    }

    private revise(views: readonly View[]): void {
        for (const listener of this.revisions) {
            listener(views);
        }
    }

    private opened(frame: EngineStateFrame): void {
        const said = frame.fit?.verdict;
        const verdict: Fit = said === undefined || said === '' ? 'engine-older' : said;
        if (verdict !== 'fits') {
            this.close();
        }
        this.since = frame.epoch === undefined ? undefined : { epoch: frame.epoch, rev: frame.rev };
        this.kept = undefined;
        this.judge(verdict);
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
