import type { EngineSliceOut, EngineStateFrame, Request } from '@jterrazz/station-api';

import { PAGE_REVISION, STATE_REVISION } from '../../../domain/fit';
import { isRecord, keyOf } from '../../../domain/request-key';
import type { Receiver } from '../engine-link/transport';

// value: the call whose recorded answer each view of engine.state is: the story `engine/state/views-equal-calls`
// value: proves a view equals its call's last answer but for when it was read, so a world's recordings answer both.
const VIEW_CALLS: Readonly<Record<string, Record<string, unknown>>> = {
    'machines.list': { in: {}, watch: 'machines.list' },
    'overview.read': { in: {}, watch: 'overview.read' },
    'overview.read+elsewhere': { in: { Elsewhere: true }, watch: 'overview.read' },
};

export function isState(request: Request): boolean {
    return 'watch' in request && request.watch === 'engine.state';
}

function keysOf(request: Request): readonly string[] {
    const input: unknown = 'in' in request ? request.in : undefined;
    if (!isRecord(input) || !Array.isArray(input.keys)) {
        return [];
    }
    return input.keys.filter((key): key is string => typeof key === 'string');
}

function lastOut(answers: readonly unknown[] | undefined): unknown {
    const outs = (answers ?? []).filter((frame) => isRecord(frame) && 'out' in frame);
    const last = outs.at(-1);
    return isRecord(last) ? last.out : undefined;
}

// value: each asked view the world holds an answer for, as the engine streams it; a view whose call the pace leaves
// value: unread is not sent, as an engine still reading it does not.
export function viewsOf(
    reads: ReadonlyMap<string, readonly unknown[]>,
    request: Request,
    unread: string | undefined,
): Record<string, EngineSliceOut> {
    const slices: Record<string, EngineSliceOut> = {};
    for (const key of keysOf(request)) {
        const asked = VIEW_CALLS[key];
        if (asked === undefined || asked.watch === unread) {
            continue;
        }
        const value = lastOut(reads.get(keyOf(asked)));
        if (value !== undefined) {
            slices[key] = { state: 'fresh', value };
        }
    }
    return slices;
}

// value: the opening and first snapshot of a world's engine.state, held open: it fits, the revision the tree speaks.
export function openingOf(
    world: string,
    slices: Record<string, EngineSliceOut>,
): { frames: readonly { v: 2; out: EngineStateFrame }[]; gap: 0; held: true } {
    const frames: readonly { v: 2; out: EngineStateFrame }[] = [
        {
            out: {
                epoch: `replay of ${world}`,
                event: 'opened',
                fit: { app: STATE_REVISION, engine: STATE_REVISION, verdict: 'fits' },
                page: PAGE_REVISION,
                rev: 1,
                state: STATE_REVISION,
                stream: 'station.state/v1',
            },
            v: 2,
        },
        { out: { event: 'snapshot', rev: 1, slices }, v: 2 },
    ];
    return { frames, gap: 0, held: true };
}

// invariant: the engine.state streams a replay holds open: a gesture that changes a read changes the views drawn from
// invariant: it, sent to every held stream as one change once the gesture's frame is handed on, as the engine's readers
// invariant: read it after the gesture; a stream held is played once its snapshot is sent, so `open` leaves it out.
export class HeldStates {
    private readonly held = new Map<Receiver, Request>();
    private rev = 1;

    get size(): number {
        return this.held.size;
    }

    hold(receiver: Receiver, request: Request): void {
        this.held.set(receiver, request);
    }

    release(receiver: Receiver): void {
        this.held.delete(receiver);
    }

    clear(): void {
        this.held.clear();
    }

    push(reads: ReadonlyMap<string, readonly unknown[]>, unread: string | undefined): void {
        setTimeout(() => {
            this.rev += 1;
            for (const [receiver, request] of this.held) {
                const slices = viewsOf(reads, request, unread);
                receiver.frame({
                    id: request.id,
                    out: { event: 'changed', rev: this.rev, slices },
                    v: 2,
                });
            }
        }, 0);
    }
}
