import type { CallName, EngineEventsOutput, EngineHello } from '@jterrazz/station-api';

import type { EnginePort, Watch } from '../ports/engine.port';
import type { PresencePort } from '../ports/presence.port';

export type Held = { open: boolean; hello: EngineHello | undefined; subscribed: boolean };

export function sameEngine(
    before: EngineHello | undefined,
    after: EngineHello | undefined,
): boolean {
    return before?.pid === after?.pid && before?.started_at === after?.started_at;
}

export function offers(hello: EngineHello | undefined, call: CallName): boolean {
    return hello?.calls?.includes(call) === true;
}

function offersEvents(hello: EngineHello | undefined): boolean {
    return offers(hello, 'engine.events');
}

function isNotice(out: EngineEventsOutput): boolean {
    return 'event' in out && out.event === 'notice';
}

// invariant: while the link is open and a component holds a read, the reads are asked again on each notice of
// invariant: engine.events the window is shown for, where the engine's hello offers it; a notice names no read, so
// invariant: it asks them all. Nothing else asks them again: no timer, and no focus.
export class Freshness {
    private readonly client: EnginePort;
    private readonly presence: PresencePort;
    private readonly refresh: () => void;
    private events: undefined | { watch: Watch; of: EngineHello | undefined };

    constructor(client: EnginePort, presence: PresencePort, refresh: () => void) {
        this.client = client;
        this.presence = presence;
        this.refresh = refresh;
    }

    update({ hello, open, subscribed }: Held): void {
        this.listen(open && subscribed && offersEvents(hello) ? hello : undefined);
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
                        this.refresh();
                    }
                },
            },
        );
        this.events = { of: hello, watch };
    }
}
