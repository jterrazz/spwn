import { isRecord, keyOf } from '../../../domain/request-key';
import type { Receiver } from '../engine-link/transport';

type Read = { request: Record<string, unknown>; answers: readonly unknown[] };

export type Played = {
    frames: readonly unknown[];
    gap: number;
    first: number;
    onFirst?: () => void;
    onLast?: (() => void) | undefined;
    // value: whether the exchange stays open once its frames are played, as a gesture held in flight does.
    held?: boolean;
};

function withId(frame: unknown, id: string): unknown {
    return isRecord(frame) ? { ...frame, id } : frame;
}

export function play(receiver: Receiver, id: string, played: Played) {
    const { frames, gap, first, onFirst, onLast, held } = played;
    const timers: ReturnType<typeof setTimeout>[] = [];
    const last = frames.length - 1;
    for (const [index, frame] of frames.entries()) {
        timers.push(
            setTimeout(
                () => {
                    if (index === 0) {
                        onFirst?.();
                    }
                    if (index === last) {
                        onLast?.();
                    }
                    receiver.frame(withId(frame, id));
                },
                first + index * gap,
            ),
        );
    }
    if (held === true) {
        return timers;
    }
    timers.push(
        setTimeout(
            () => {
                if (last < 0) {
                    onLast?.();
                }
                receiver.end('answered', true);
            },
            first + frames.length * gap,
        ),
    );
    return timers;
}

// invariant: a gesture's reads answer as the world answered them while it ran from its first frame, and once its
// invariant: last is played, as they answered after it: a read it changed as recorded then, any other as before it.
export function inFlight(
    reads: Map<string, readonly unknown[]>,
    during: readonly Read[],
    after: readonly Read[],
): { begun: () => void; ended: () => void } {
    const before = new Map<string, readonly unknown[] | undefined>();
    return {
        begun: () => {
            for (const read of during) {
                const key = keyOf(read.request);
                before.set(key, reads.get(key));
                reads.set(key, read.answers);
            }
        },
        ended: () => {
            for (const [key, answers] of before) {
                if (answers === undefined) {
                    reads.delete(key);
                } else {
                    reads.set(key, answers);
                }
            }
            for (const read of after) {
                reads.set(keyOf(read.request), read.answers);
            }
        },
    };
}
