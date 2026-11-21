import type { Request } from '@jterrazz/station-api';

import { isRecord } from '../../../domain/request-key';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';

// value: the first out of the engine's overview.read, every line pending: no need read yet, and a verdict that says so.
function pendingOf(out: unknown): unknown {
    const summary = isRecord(out) ? out.Summary : undefined;
    const head = isRecord(summary) ? summary.Head : undefined;
    if (!isRecord(out) || !isRecord(summary) || !isRecord(head)) {
        return out;
    }
    const name = String(head.Name);
    return {
        ...out,
        Summary: {
            ...summary,
            Needs: [],
            Pending: true,
            Verdict: { Line: '', Sentence: `Reading ${name}.`, Tone: 'plain' },
        },
    };
}

// value: how long the engine takes from an overview's first out to the rest, long enough for a frame to be drawn.
const READ_FOR = 100;

// value: `overviews`, the overview.read streams begun; `answered`, those whose last out was handed on before their end.
// value: a transport that ends every watch as the engine's socket does, its connection closed after its last out with
// value: no done, and that answers overview.read first with every line pending, as the engine composes it.
export function socketEnded(transport: Transport) {
    let overviews = 0;
    let answered = 0;
    return {
        answered: () => answered,
        overviews: () => overviews,
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) => {
                if (!('watch' in request)) {
                    return transport.open(request, receiver);
                }
                const overview = request.watch === 'overview.read';
                let first = true;
                let ended = false;
                return transport.open(request, {
                    end: (reason, sent) => {
                        const pass = () => {
                            if (!ended) {
                                ended = true;
                                receiver.end(reason, sent);
                            }
                        };
                        if (overview) {
                            setTimeout(pass, READ_FOR);
                        } else {
                            pass();
                        }
                    },
                    frame: (frame) => {
                        const pass = () => {
                            if (ended) {
                                return;
                            }
                            if (isRecord(frame) && 'done' in frame) {
                                ended = true;
                                answered += overview ? 1 : 0;
                                receiver.end('the engine closed the connection', true);
                                return;
                            }
                            receiver.frame(frame);
                        };
                        if (overview && first && isRecord(frame) && 'out' in frame) {
                            overviews += 1;
                            receiver.frame({ ...frame, out: pendingOf(frame.out) });
                        }
                        first = false;
                        if (overview) {
                            setTimeout(pass, READ_FOR);
                        } else {
                            pass();
                        }
                    },
                });
            },
        } satisfies Transport,
    };
}
