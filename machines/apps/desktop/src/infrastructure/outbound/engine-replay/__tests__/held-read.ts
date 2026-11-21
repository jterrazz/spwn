import type { Request } from '@jterrazz/station-api';

import { isRecord } from '../../../../domain/request-key';
import type { Exchange, Receiver, Transport } from '../../engine-link/transport';
import type { Stage } from '../replay';

type Withheld = { id: string; receiver: Receiver; rev: number; slices: Record<string, unknown> };

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

function isViewOf(key: string, name: string): boolean {
    return key === name || key.startsWith(`${name}+`);
}

// value: an engine.state stream whose snapshot sends the views of `name` unread, as an engine still reading them,
// value: and keeps them for the release.
function withholding(
    name: string,
    request: Request,
    receiver: Receiver,
    kept: Withheld[],
): Receiver {
    return {
        end: receiver.end,
        frame: (frame) => {
            const out = isRecord(frame) && isRecord(frame.out) ? frame.out : undefined;
            if (out?.event !== 'snapshot' || !isRecord(out.slices)) {
                receiver.frame(frame);
                return;
            }
            const entries = Object.entries(out.slices);
            const views = entries.filter(([key]) => isViewOf(key, name));
            if (views.length === 0) {
                receiver.frame(frame);
                return;
            }
            const unread = entries.map(([key, slice]): [string, unknown] => [
                key,
                isViewOf(key, name) ? { state: 'unread' } : slice,
            ]);
            kept.push({
                id: request.id,
                receiver,
                rev: Number(out.rev),
                slices: Object.fromEntries(views),
            });
            receiver.frame({
                id: request.id,
                out: { ...out, slices: Object.fromEntries(unread) },
                v: 2,
            });
        },
    };
}

// value: a stage whose requests of one name, call or watch, wait unanswered until released, then play as recorded;
// value: the views engine.state draws from that name are sent unread until then, and read at the release.
export function holdingRead(stage: Stage, name: string): HeldRead {
    const queued: Queued[] = [];
    const withheld: Withheld[] = [];
    let released = false;
    return {
        held: () =>
            queued.filter((one) => !one.closed && one.opened === undefined).length +
            withheld.length,
        release: () => {
            released = true;
            for (const one of queued.filter((each) => !each.closed)) {
                one.opened = stage.transport.open(one.request, one.receiver);
            }
            for (const { id, receiver, rev, slices } of withheld.splice(0)) {
                receiver.frame({ id, out: { event: 'changed', rev: rev + 1, slices }, v: 2 });
            }
        },
        transport: {
            link: (listener) => stage.transport.link(listener),
            open: (request, receiver) => {
                if (!released && nameOf(request) === 'engine.state') {
                    return stage.transport.open(
                        request,
                        withholding(name, request, receiver, withheld),
                    );
                }
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
