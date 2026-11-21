import type { Request } from '@jterrazz/station-api';

import { isRecord } from '../../../domain/request-key';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';

// value: how long the engine takes from a stream's snapshot to the views it was still reading, and from them to the
// value: end of its connection, long enough for a frame to be drawn between each.
const READ_FOR = 100;

// value: a snapshot whose Overview views are not read yet: sent without their value, as an engine still reading them.
function unreadOf(out: Record<string, unknown>): Record<string, unknown> {
    const slices = isRecord(out.slices) ? out.slices : {};
    const unread = Object.fromEntries(
        Object.entries(slices).map(([key, slice]) =>
            key.startsWith('overview.read') ? [key, { state: 'unread' }] : [key, slice],
        ),
    );
    return { ...out, slices: unread };
}

function snapshotOf(frame: unknown): Record<string, unknown> | undefined {
    const out = isRecord(frame) && isRecord(frame.out) ? frame.out : undefined;
    return out?.event === 'snapshot' ? out : undefined;
}

// value: an engine.state stream whose snapshot sends the Overview views unread, then the views it read, then ends its
// value: connection with no done; `answered` counts those whose whole views were handed on before their end.
function stateEnded(
    request: Request,
    receiver: Receiver,
    transport: Transport,
    answered: () => void,
) {
    let ended = false;
    const timers: ReturnType<typeof setTimeout>[] = [];
    const exchange = transport.open(request, {
        end: (reason, sent) => {
            if (!ended) {
                ended = true;
                receiver.end(reason, sent);
            }
        },
        frame: (frame) => {
            const snapshot = snapshotOf(frame);
            if (ended || snapshot === undefined || !isRecord(frame)) {
                receiver.frame(frame);
                return;
            }
            receiver.frame({ ...frame, out: unreadOf(snapshot) });
            const rev = Number(snapshot.rev) + 1;
            timers.push(
                setTimeout(() => {
                    receiver.frame({ ...frame, out: { ...snapshot, event: 'changed', rev } });
                    answered();
                }, READ_FOR),
                setTimeout(() => {
                    ended = true;
                    receiver.end('the engine closed the connection', true);
                }, 2 * READ_FOR),
            );
        },
    });
    return {
        cancel: exchange.cancel,
        close: () => {
            for (const timer of timers) {
                clearTimeout(timer);
            }
            exchange.close();
        },
    };
}

// value: a transport that ends every watch as the engine's socket does, its connection closed after its last out with
// value: no done, and whose engine.state streams each answer the Overview views unread before whole.
export function socketEnded(transport: Transport) {
    let answered = 0;
    return {
        answered: () => answered,
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) => {
                if (!('watch' in request)) {
                    return transport.open(request, receiver);
                }
                if (request.watch === 'engine.state') {
                    return stateEnded(request, receiver, transport, () => {
                        answered += 1;
                    });
                }
                let ended = false;
                return transport.open(request, {
                    end: (reason, sent) => {
                        if (!ended) {
                            ended = true;
                            receiver.end(reason, sent);
                        }
                    },
                    frame: (frame) => {
                        if (ended) {
                            return;
                        }
                        if (isRecord(frame) && 'done' in frame) {
                            ended = true;
                            receiver.end('the engine closed the connection', true);
                            return;
                        }
                        receiver.frame(frame);
                    },
                });
            },
        } satisfies Transport,
    };
}
