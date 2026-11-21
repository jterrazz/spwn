import type { Request } from '@jterrazz/station-api';

import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';

const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'macbook-pro did not answer',
    type: 'unreachable',
};

// value: a transport that counts the frames it hands on, and refuses as unreachable every call named by `refuse`.
export function counted(transport: Transport) {
    let frames = 0;
    let refused = 0;
    let refusing: string | undefined;
    const refusal = (request: Request, receiver: Receiver) => {
        const timer = setTimeout(() => {
            refused += 1;
            receiver.frame({ error: UNREACHABLE, id: request.id, v: 2 });
        }, 0);
        return {
            cancel: () => undefined,
            close: () => {
                clearTimeout(timer);
            },
        };
    };
    return {
        frames: () => frames,
        refuse: (call: string) => {
            refusing = call;
        },
        refused: () => refused,
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) =>
                'call' in request && request.call === refusing
                    ? refusal(request, receiver)
                    : transport.open(request, {
                          end: (reason, sent) => {
                              receiver.end(reason, sent);
                          },
                          frame: (frame) => {
                              frames += 1;
                              receiver.frame(frame);
                          },
                      }),
        } satisfies Transport,
    };
}
