import type { Request } from '@jterrazz/station-api';

import { isRecord } from '../../../domain/request-key';
import type { Receiver, Transport } from '../../../infrastructure/outbound/engine-link/transport';

type Slices = Record<string, unknown>;

function snapshotOf(frame: unknown): Record<string, unknown> | undefined {
    const out = isRecord(frame) && isRecord(frame.out) ? frame.out : undefined;
    return out?.event === 'snapshot' ? out : undefined;
}

// value: a transport whose engine.state stream a test pushes revisions to: `push`, the views as `edit` makes them
// value: from those last sent; `same`, a revision whose views equal those last sent.
export function pushed(transport: Transport) {
    let held: undefined | { id: string; receiver: Receiver };
    let slices: Slices = {};
    let rev = 1;
    const push = (edit: (views: Slices) => Slices) => {
        if (held === undefined) {
            throw new Error('engine.state is not held');
        }
        rev += 1;
        slices = edit(structuredClone(slices));
        held.receiver.frame({ id: held.id, out: { event: 'changed', rev, slices }, v: 2 });
    };
    return {
        push,
        same: () => {
            push((views) => views);
        },
        transport: {
            link: (listener: Parameters<Transport['link']>[0]) => transport.link(listener),
            open: (request: Request, receiver: Receiver) => {
                if (!('watch' in request) || request.watch !== 'engine.state') {
                    return transport.open(request, receiver);
                }
                held = { id: request.id, receiver };
                return transport.open(request, {
                    end: receiver.end,
                    frame: (frame) => {
                        const { slices: sent, rev: at } = snapshotOf(frame) ?? {};
                        if (isRecord(sent)) {
                            slices = sent;
                            rev = Number(at);
                        }
                        receiver.frame(frame);
                    },
                });
            },
        } satisfies Transport,
    };
}

// value: the views with both Overviews' verdict saying `sentence`, every other view as it was.
export function said(sentence: string) {
    return (views: Record<string, unknown>) =>
        Object.fromEntries(
            Object.entries(views).map(([key, view]) => {
                const value = isRecord(view) && isRecord(view.value) ? view.value : undefined;
                const summary = isRecord(value?.Summary) ? value.Summary : undefined;
                if (!key.startsWith('overview.read') || !isRecord(view) || !summary) {
                    return [key, view];
                }
                const verdict = {
                    ...(isRecord(summary.Verdict) ? summary.Verdict : {}),
                    Sentence: sentence,
                };
                return [
                    key,
                    { ...view, value: { ...value, Summary: { ...summary, Verdict: verdict } } },
                ];
            }),
        );
}
