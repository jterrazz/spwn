import type { EngineSliceOut, EngineStateFrame, Request } from '@jterrazz/station-api';

import { PAGE_REVISION, STATE_REVISION } from '../../../domain/fit';
import { isRecord } from '../../../domain/request-key';
import { askedBy, callOfView } from '../../../domain/views';
import type { Receiver } from '../engine-link/transport';
import type { Answers } from './answers';

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

// value: each asked view the world holds an answer for, as the engine streams it: the story
// value: `engine/state/views-equal-calls*` proves a view equals its call's last answer but for when it was read, so a
// value: world's recordings answer both. A view whose call the pace leaves unread is not sent, as an engine still
// value: reading it does not.
export function viewsOf(
    answers: Pick<Answers, 'reads' | 'views'>,
    request: Request,
    unread: string | undefined,
): Record<string, EngineSliceOut> {
    const keys = keysOf(request);
    const slices: Record<string, EngineSliceOut> = {};
    for (const [key, recorded] of answers.reads) {
        const view = answers.views.get(key);
        if (
            view === undefined ||
            callOfView(view) === unread ||
            !keys.some((asked) => askedBy(asked, view))
        ) {
            continue;
        }
        const value = lastOut(recorded);
        if (value !== undefined) {
            slices[view] = { state: 'fresh', value };
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

    push(answers: Pick<Answers, 'reads' | 'views'>, unread: string | undefined): void {
        setTimeout(() => {
            this.rev += 1;
            for (const [receiver, request] of this.held) {
                const slices = viewsOf(answers, request, unread);
                receiver.frame({
                    id: request.id,
                    out: { event: 'changed', rev: this.rev, slices },
                    v: 2,
                });
            }
        }, 0);
    }
}
