import type { Request } from '@jterrazz/station-api';

import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';

const UNREACHABLE = {
    code: 'remote.unreachable',
    message: 'macbook-pro did not answer',
    type: 'unreachable',
};

const NOTICE = { event: 'notice', notice: { kind: 'vm_ready', text: 'linux-1 is ready' } };

// value: a transport that counts the frames it hands on, refuses as unreachable every call named by `refuse`, and
// value: answers engine.events itself, with a notice on each `notice`.
export function counted(transport: Transport) {
    let frames = 0;
    let refused = 0;
    let refusing: string | undefined;
    const events = new Map<Receiver, Request>();
    const listening = (request: Request, receiver: Receiver) => {
        events.set(receiver, request);
        return {
            cancel: () => undefined,
            close: () => {
                events.delete(receiver);
            },
        };
    };
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
        notice: () => {
            for (const [receiver, request] of events) {
                receiver.frame({ id: request.id, out: NOTICE, v: 2 });
            }
        },
        refuse: (call: string) => {
            refusing = call;
        },
        refused: () => refused,
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) => {
                if ('watch' in request && request.watch === 'engine.events') {
                    return listening(request, receiver);
                }
                return 'call' in request && request.call === refusing
                    ? refusal(request, receiver)
                    : transport.open(request, {
                          end: (reason, sent) => {
                              receiver.end(reason, sent);
                          },
                          frame: (frame) => {
                              frames += 1;
                              receiver.frame(frame);
                          },
                      });
            },
        } satisfies Transport,
    };
}
