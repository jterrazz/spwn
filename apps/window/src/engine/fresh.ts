import type { EngineEventsOutput, EngineHello } from '@jterrazz/station-api';

import type { Client, Watch } from './client';
import type { Presence } from './presence';

// value: how often every read a component shows is asked again while the window is shown. The engine reads its own
// value: pages (machines, ports, accounts, needs) every 30 s, so a window asking more often reads the same answer.
export const FRESH_EVERY = 30_000;

export type Why = 'back' | 'notice' | 'tick';

export type Held = { open: boolean; hello: EngineHello | undefined; subscribed: boolean };

export function sameEngine(
    before: EngineHello | undefined,
    after: EngineHello | undefined,
): boolean {
    return before?.pid === after?.pid && before?.started_at === after?.started_at;
}

function offersEvents(hello: EngineHello | undefined): boolean {
    return hello?.calls?.includes('engine.events') === true;
}

function isNotice(out: EngineEventsOutput): boolean {
    return 'event' in out && out.event === 'notice';
}

type Timing = { every: number };

// invariant: while the link is open and a component holds a read, the reads are asked again on one cadence while the
// invariant: window is shown, at once when it is shown again or regains the focus, and on each notice of
// invariant: engine.events where the engine's hello offers it; a notice names no read, so it asks them all.
export class Freshness {
    private readonly client: Client;
    private readonly presence: Presence;
    private readonly timing: Timing;
    private readonly refresh: (why: Why) => void;
    private held: Held = { hello: undefined, open: false, subscribed: false };
    private timer: ReturnType<typeof setInterval> | undefined;
    private events: undefined | { watch: Watch; of: EngineHello | undefined };
    private unpresence: (() => void) | undefined;

    constructor(client: Client, presence: Presence, timing: Timing, refresh: (why: Why) => void) {
        this.client = client;
        this.presence = presence;
        this.timing = timing;
        this.refresh = refresh;
    }

    update(held: Held): void {
        this.held = held;
        this.settle();
    }

    private settle(): void {
        const { hello, open, subscribed } = this.held;
        const active = open && subscribed;
        if (active) {
            this.unpresence ??= this.presence.subscribe((seen) => {
                this.seen(seen);
            });
        } else {
            this.unpresence?.();
            this.unpresence = undefined;
        }
        this.tick(active && this.presence.visible());
        this.listen(active && offersEvents(hello) ? hello : undefined);
    }

    private tick(ticking: boolean): void {
        if (ticking && this.timer === undefined) {
            this.timer = setInterval(() => {
                this.refresh('tick');
            }, this.timing.every);
        } else if (!ticking) {
            clearInterval(this.timer);
            this.timer = undefined;
        }
    }

    private listen(hello: EngineHello | undefined): void {
        if (this.events && hello && sameEngine(this.events.of, hello)) {
            return;
        }
        this.events?.watch.close();
        this.events = undefined;
        if (hello === undefined) {
            return;
        }
        const watch = this.client.watch(
            'engine.events',
            {},
            {
                error: () => undefined,
                out: (out) => {
                    if (isNotice(out) && this.presence.visible()) {
                        this.refresh('notice');
                    }
                },
            },
        );
        this.events = { of: hello, watch };
    }

    private seen(seen: 'back' | 'hidden'): void {
        if (seen === 'back' && this.presence.visible()) {
            this.tick(false);
            this.refresh('back');
        }
        this.settle();
    }
}
