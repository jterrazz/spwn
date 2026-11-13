import type { CallName } from '@jterrazz/station-api';

import type { Client } from './client';
import { failed, failedOf, flush, READING, show, stop } from './entry';
import type { Asked, Entry, Reading } from './entry';
import { keyOf } from './frame';
import { FRESH_EVERY, Freshness, sameEngine } from './fresh';
import type { Why } from './fresh';
import { documentPresence } from './presence';
import type { Presence } from './presence';
import type { Link } from './transport';

export type StoreTiming = { grace: number; every: number };

const DEFAULTS: StoreTiming = { every: FRESH_EVERY, grace: 1500 };

export function askedKey(asked: Asked): string {
    const verb = asked.kind === 'read' ? 'call' : 'watch';
    return keyOf({ [verb]: asked.name, in: asked.input, target: asked.target });
}

// invariant: one entry per request compared by value (its call, input and target), shared by every subscriber:
// invariant: one socket each, the last answer kept while one remains, closed a grace after the last leaves. Every
// invariant: entry is asked again when the link opens, or opens on another engine (its hello's pid or start).
// invariant: An entry a component holds is asked again quietly by `Freshness` once its answer has settled: its
// invariant: answer stays shown, and a watch's next outs are held until it ends or the next tick, so a re-ask
// invariant: never draws a pending answer over a settled one; one still asking is never asked twice.
export class ReadStore {
    private readonly client: Client;
    private readonly timing: StoreTiming;
    private readonly entries = new Map<string, Entry>();
    private readonly linkListeners = new Set<() => void>();
    private readonly freshness: Freshness;
    private shown: Link | undefined;
    private unlink: (() => void) | undefined;

    constructor(
        client: Client,
        timing: Partial<StoreTiming> = {},
        presence: Presence = documentPresence(),
    ) {
        this.client = client;
        this.timing = { ...DEFAULTS, ...timing };
        this.freshness = new Freshness(client, presence, this.timing, (why) => {
            this.refresh(why);
        });
    }

    readonly reading = (key: string): Reading<unknown> => this.entries.get(key)?.reading ?? READING;

    readonly link = (): Link | undefined => this.shown;

    readonly subscribe = (asked: Asked, listener: () => void): (() => void) => {
        const key = askedKey(asked);
        const entry = this.entryOf(key, asked);
        clearTimeout(entry.grace);
        entry.grace = undefined;
        entry.listeners.add(listener);
        this.fresh();
        return () => {
            entry.listeners.delete(listener);
            if (entry.listeners.size === 0) {
                entry.grace = setTimeout(() => {
                    this.forget(key, entry);
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
            if (names.includes(entry.asked.name)) {
                this.ask(entry, false);
            }
        }
    };

    private refresh(why: Why): void {
        for (const entry of this.entries.values()) {
            if (entry.listeners.size === 0) {
                continue;
            }
            if (!entry.asking) {
                this.ask(entry, true);
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
            held: undefined,
            listeners: new Set(),
            reading: READING,
            subscribe: (listener) => this.subscribe(asked, listener),
            watch: undefined,
        };
        this.entries.set(key, entry);
        this.hold();
        if (this.shown?.kind === 'open') {
            this.ask(entry, false);
        }
        return entry;
    }

    private forget(key: string, entry: Entry): void {
        if (entry.listeners.size > 0) {
            return;
        }
        stop(entry);
        this.entries.delete(key);
        this.release();
    }

    private hold(): void {
        this.unlink ??= this.client.link((link) => {
            this.heard(link);
        });
    }

    private release(): void {
        if (this.entries.size === 0 && this.linkListeners.size === 0) {
            this.unlink?.();
            this.unlink = undefined;
            this.shown = undefined;
            this.fresh();
        }
    }

    private heard(link: Link): void {
        const before = this.shown;
        this.shown = link;
        for (const listener of this.linkListeners) {
            listener();
        }
        const opened = link.kind === 'open';
        const wasOpen = before?.kind === 'open';
        if (opened && (!wasOpen || !sameEngine(before.hello, link.hello))) {
            for (const entry of this.entries.values()) {
                this.ask(entry, false);
            }
        } else if (!opened && wasOpen) {
            for (const entry of this.entries.values()) {
                stop(entry);
            }
        }
        this.fresh();
    }

    private ask(entry: Entry, quiet: boolean): void {
        stop(entry);
        entry.asking = true;
        const { asked, asks } = entry;
        const current = () => entry.asks === asks;
        if (asked.kind === 'read') {
            void this.read(entry, asked, current);
            return;
        }
        const holds = quiet && entry.reading.kind === 'read';
        let last: undefined | { out: unknown };
        entry.watch = this.client.watch(
            asked.name,
            asked.input,
            {
                done: () => {
                    if (current()) {
                        entry.asking = false;
                        entry.held = undefined;
                        if (last) {
                            show(entry, { done: true, kind: 'read', out: last.out });
                        }
                    }
                },
                error: (error) => {
                    if (current()) {
                        entry.asking = error.wire === undefined;
                        failed(entry, error);
                    }
                },
                out: (out: unknown) => {
                    if (current()) {
                        last = { out };
                        if (holds) {
                            entry.held = last;
                        } else {
                            show(entry, { done: false, kind: 'read', out });
                        }
                    }
                },
            },
            asked.target,
        );
    }

    private async read(
        entry: Entry,
        asked: Extract<Asked, { kind: 'read' }>,
        current: () => boolean,
    ): Promise<void> {
        try {
            const out: unknown = await this.client.call(
                asked.name,
                asked.input,
                asked.target ? { target: asked.target } : {},
            );
            if (current()) {
                entry.asking = false;
                show(entry, { done: true, kind: 'read', out });
            }
        } catch (error) {
            if (current()) {
                entry.asking = false;
                failed(entry, failedOf(error));
            }
        }
    }
}
