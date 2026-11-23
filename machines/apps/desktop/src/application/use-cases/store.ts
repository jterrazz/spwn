import type { CallName } from '@jterrazz/station-api';

import { keyOf } from '../../domain/request-key';
import type { EnginePort, Link } from '../ports/engine.port';
import type { PresencePort } from '../ports/presence.port';
import { ask } from './ask';
import { flush, READING, stop } from './entry';
import type { Asked, Entry, Reading } from './entry';
import { FRESH_EVERY, Freshness, sameEngine } from './fresh';
import type { Why } from './fresh';
import type { StateMirror } from './mirror';
import { judgedOf } from './reach';
import type { Judged } from './reach';

// value: `grace`, how long a request nobody holds stays open; `keptFor` and `keptAtMost`, how long and how many of the
// value: answers nobody holds are kept after it, so a page drawn again shows its answer at once; `now`, the clock an
// value: answer's instant is read off.
export type StoreTiming = {
    grace: number;
    every: number;
    keptFor: number;
    keptAtMost: number;
    now: () => Date;
};

const DEFAULTS: StoreTiming = {
    every: FRESH_EVERY,
    grace: 1500,
    keptAtMost: 50,
    keptFor: 600_000,
    now: () => new Date(),
};

export function askedKey(asked: Asked): string {
    const verb = asked.kind === 'read' ? 'call' : 'watch';
    return keyOf({ [verb]: asked.name, in: asked.input, target: asked.target });
}

// invariant: one entry per request compared by value (its call, input and target), shared by every subscriber:
// invariant: one socket each, closed a grace after the last leaves; its answer is then kept, the request stopped,
// invariant: until `keptFor` has passed or `keptAtMost` answers left later are kept, and a subscriber back meanwhile
// invariant: sees it at once while it is asked again. Every entry still open is asked again when the link opens, or
// invariant: opens on another engine (its hello's pid or start); a kept one is asked when it is held again. A link
// invariant: the mirror judges incompatible is no open link: nothing is asked of its engine until another that fits.
// invariant: An entry a component holds is asked again by `Freshness` once its answer has settled, and every ask
// invariant: over a settled answer (the cadence's, a gesture's rereads, a relink's) is quiet: the answer stays shown,
// invariant: and a watch's next outs are held until it ends or the next tick, so a re-ask never draws a pending
// invariant: answer over a settled one; one still asking is never asked twice by the cadence.
export class ReadStore {
    private readonly client: EnginePort;
    private readonly timing: StoreTiming;
    private readonly entries = new Map<string, Entry>();
    private readonly linkListeners = new Set<() => void>();
    private readonly freshness: Freshness;
    private readonly judging: Pick<StateMirror, 'fitOf' | 'subscribeFit'>;
    private heardLink: Link | undefined;
    private shown: Judged | undefined;
    private unlink: (() => void) | undefined;
    private unjudge: (() => void) | undefined;

    constructor(
        client: EnginePort,
        presence: PresencePort,
        judging: Pick<StateMirror, 'fitOf' | 'subscribeFit'>,
        timing: Partial<StoreTiming> = {},
    ) {
        this.client = client;
        this.judging = judging;
        this.timing = { ...DEFAULTS, ...timing };
        this.freshness = new Freshness(client, presence, this.timing, (why) => {
            this.refresh(why);
        });
    }

    readonly reading = (key: string): Reading<unknown> => this.entries.get(key)?.reading ?? READING;

    readonly link = (): Judged | undefined => this.shown;

    readonly subscribe = (asked: Asked, listener: () => void): (() => void) => {
        const key = askedKey(asked);
        const entry = this.entryOf(key, asked);
        clearTimeout(entry.grace);
        entry.grace = undefined;
        entry.listeners.add(listener);
        if (entry.kept) {
            entry.kept = false;
            if (this.shown?.kind === 'open') {
                ask(this.client, entry);
            }
        }
        this.fresh();
        return () => {
            entry.listeners.delete(listener);
            if (entry.listeners.size === 0) {
                entry.grace = setTimeout(() => {
                    this.keep(key, entry);
                }, this.timing.grace);
            }
            this.fresh();
        };
    };

    // value: the same function for every request equal to one subscribed, so React keeps its subscription.
    readonly subscriberOf = (asked: Asked): ((listener: () => void) => () => void) =>
        this.entries.get(askedKey(asked))?.subscribe ??
        ((listener) => this.subscribe(asked, listener));

    readonly subscribeLink = (listener: () => void): (() => void) => {
        this.linkListeners.add(listener);
        this.hold();
        return () => {
            this.linkListeners.delete(listener);
            this.release();
        };
    };

    readonly reread = (names: readonly CallName[]): void => {
        if (this.shown?.kind !== 'open') {
            return;
        }
        for (const entry of this.entries.values()) {
            if (!entry.kept && names.includes(entry.asked.name)) {
                ask(this.client, entry);
            }
        }
    };

    private refresh(why: Why): void {
        for (const entry of this.entries.values()) {
            if (entry.listeners.size === 0) {
                continue;
            }
            if (!entry.asking) {
                ask(this.client, entry);
            } else if (why === 'tick') {
                flush(entry);
            }
        }
    }

    private fresh(): void {
        const open = this.shown?.kind === 'open';
        this.freshness.update({
            hello: this.shown?.kind === 'open' ? this.shown.hello : undefined,
            open,
            subscribed: [...this.entries.values()].some((entry) => entry.listeners.size > 0),
        });
    }

    private entryOf(key: string, asked: Asked): Entry {
        const existing = this.entries.get(key);
        if (existing) {
            return existing;
        }
        const entry: Entry = {
            asked,
            asking: false,
            asks: 0,
            grace: undefined,
            heardAt: undefined,
            held: undefined,
            kept: false,
            key,
            listeners: new Set(),
            mark: undefined,
            now: this.timing.now,
            reading: READING,
            subscribe: (listener) => this.subscribe(asked, listener),
            watch: undefined,
        };
        this.entries.set(key, entry);
        this.hold();
        if (this.shown?.kind === 'open') {
            ask(this.client, entry);
        }
        return entry;
    }

    private keep(key: string, entry: Entry): void {
        if (entry.listeners.size > 0) {
            return;
        }
        stop(entry);
        entry.kept = true;
        this.entries.delete(key);
        this.entries.set(key, entry);
        entry.grace = setTimeout(() => {
            this.forget(key, entry);
        }, this.timing.keptFor);
        const kept = [...this.entries].filter(([, one]) => one.kept);
        for (const [oldest, one] of kept.slice(
            0,
            Math.max(0, kept.length - this.timing.keptAtMost),
        )) {
            this.forget(oldest, one);
        }
    }

    private forget(key: string, entry: Entry): void {
        if (entry.listeners.size > 0) {
            return;
        }
        clearTimeout(entry.grace);
        stop(entry);
        this.entries.delete(key);
        this.release();
    }

    private hold(): void {
        this.unjudge ??= this.judging.subscribeFit(() => {
            this.judge();
        });
        this.unlink ??= this.client.link((link) => {
            this.heardLink = link;
            this.judge();
        });
    }

    private release(): void {
        if (this.entries.size === 0 && this.linkListeners.size === 0) {
            this.unlink?.();
            this.unlink = undefined;
            this.unjudge?.();
            this.unjudge = undefined;
            this.heardLink = undefined;
            this.shown = undefined;
            this.fresh();
        }
    }

    private judged(): Judged | undefined {
        const heard = this.heardLink;
        if (heard?.kind !== 'open') {
            return heard;
        }
        return judgedOf(heard, this.judging.fitOf(heard.hello));
    }

    private judge(): void {
        const before = this.shown;
        const link = this.judged();
        if (link === undefined || link === before) {
            return;
        }
        this.shown = link;
        for (const listener of this.linkListeners) {
            listener();
        }
        this.moved(before, link);
        this.fresh();
    }

    private moved(before: Judged | undefined, link: Judged): void {
        const wasOpen = before?.kind === 'open';
        if (link.kind === 'open' && (!wasOpen || !sameEngine(before.hello, link.hello))) {
            for (const entry of this.entries.values()) {
                if (!entry.kept) {
                    ask(this.client, entry);
                }
            }
        } else if (link.kind !== 'open' && wasOpen) {
            for (const entry of this.entries.values()) {
                stop(entry);
            }
        }
    }
}
