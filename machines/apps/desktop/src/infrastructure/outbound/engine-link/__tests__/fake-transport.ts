import type { CancelFrame, Request } from '@jterrazz/station-api';

import type { Link } from '../../../../application/ports/engine.port';
import type { Receiver, Transport } from '../transport';

export type Opened = {
    request: Request;
    receiver: Receiver;
    closed: boolean;
    cancelled: CancelFrame[];
    sentBeforeClose: boolean;
};

export function fakeTransport() {
    const opened: Opened[] = [];
    const listeners: ((link: Link) => void)[] = [];
    const transport: Transport = {
        link: (listener) => {
            listeners.push(listener);
            return () => {
                listeners.splice(listeners.indexOf(listener), 1);
            };
        },
        open: (request, receiver) => {
            const entry: Opened = {
                cancelled: [],
                closed: false,
                receiver,
                request,
                sentBeforeClose: false,
            };
            opened.push(entry);
            return {
                cancel: (frame) => {
                    entry.cancelled.push(frame);
                    entry.sentBeforeClose = !entry.closed;
                },
                close: () => {
                    entry.closed = true;
                },
            };
        },
    };
    return { listeners, opened, transport };
}

export function last(opened: readonly Opened[]): Opened {
    const entry = opened.at(-1);
    if (!entry) {
        throw new Error('nothing was opened');
    }
    return entry;
}
