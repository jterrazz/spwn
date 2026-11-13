import type { Request } from '@jterrazz/station-api';

import type { Stage } from '../engine/replay';
import type { Receiver, Transport } from '../engine/transport';

type Held = { id: string; receiver: Receiver };

export type HeldCall = {
    transport: Transport;
    sent: () => boolean;
    progress: (task: string) => void;
    drop: () => void;
};

export function holdingCall(stage: Stage, call: string): HeldCall {
    let held: Held | undefined;
    const isHeld = (request: Request) => 'call' in request && request.call === call;
    return {
        drop: () => {
            const dropped = held;
            held = undefined;
            dropped?.receiver.end('the engine closed the connection', true);
            stage.drop();
        },
        progress: (task) => {
            held?.receiver.frame({ id: held.id, progress: { task }, v: 2 });
        },
        sent: () => held !== undefined,
        transport: {
            link: (listener) => stage.transport.link(listener),
            open: (request, receiver) => {
                if (!isHeld(request)) {
                    return stage.transport.open(request, receiver);
                }
                const opened: Held = { id: request.id, receiver };
                held = opened;
                return {
                    close: () => {
                        if (held === opened) {
                            held = undefined;
                        }
                    },
                };
            },
        },
    };
}
