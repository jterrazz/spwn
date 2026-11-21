import type { Request } from '@jterrazz/station-api';

import type { Exchange, Receiver, Transport } from '../../engine-link/transport';
import type { Stage } from '../replay';

type Queued = { request: Request; receiver: Receiver; closed: boolean; opened?: Exchange };

export type HeldRead = {
    transport: Transport;
    release: () => void;
    held: () => number;
};

function nameOf(request: Request): string {
    if ('call' in request) {
        return request.call;
    }
    return 'watch' in request ? request.watch : '';
}

// value: a stage whose requests of one name, call or watch, wait unanswered until released, then play as recorded.
export function holdingRead(stage: Stage, name: string): HeldRead {
    const queued: Queued[] = [];
    let released = false;
    return {
        held: () => queued.filter((one) => !one.closed && one.opened === undefined).length,
        release: () => {
            released = true;
            for (const one of queued.filter((each) => !each.closed)) {
                one.opened = stage.transport.open(one.request, one.receiver);
            }
        },
        transport: {
            link: (listener) => stage.transport.link(listener),
            open: (request, receiver) => {
                if (released || nameOf(request) !== name) {
                    return stage.transport.open(request, receiver);
                }
                const one: Queued = { closed: false, receiver, request };
                queued.push(one);
                return {
                    cancel: (frame) => {
                        one.opened?.cancel(frame);
                    },
                    close: () => {
                        one.closed = true;
                        one.opened?.close();
                    },
                };
            },
        },
    };
}
