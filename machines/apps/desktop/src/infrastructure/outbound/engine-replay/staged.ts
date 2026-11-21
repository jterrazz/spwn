import type { Request } from '@jterrazz/station-api';

import { isRecord } from '../../../domain/request-key';
import type { Played } from './played';

type Staged = { unread?: string | undefined; stale?: string | undefined };

type Recorded = { exchanges: readonly { answers: readonly unknown[] }[] };

function nameOf(request: Request): string {
    if ('call' in request) {
        return request.call;
    }
    return 'watch' in request ? request.watch : '';
}

function unreachable(frame: unknown): boolean {
    return isRecord(frame) && isRecord(frame.error) && frame.error.type === 'unreachable';
}

// value: the first frame of the world that refuses as an unreachable Mac does, the refusal a stale read replays.
export function refusalOf(recording: Recorded): unknown {
    return recording.exchanges.flatMap((exchange) => exchange.answers).find(unreachable);
}

// invariant: a request the pace leaves unread is held open with no frame; one it makes stale answers as recorded
// invariant: the first time its key is asked and is refused every time after.
export function stagedOf(
    request: Request,
    asked: number,
    staged: Staged | undefined,
    refusal: unknown,
): Omit<Played, 'first'> | undefined {
    const name = nameOf(request);
    if (name === staged?.unread) {
        return { frames: [], gap: 0, held: true };
    }
    if (name === staged?.stale && asked > 0 && refusal !== undefined) {
        return { frames: [refusal], gap: 0 };
    }
    return undefined;
}
