import { isRecord } from '../../../domain/request-key';
import type { Recording } from './replay';

// value: the instant a world was recorded at, as its fleet's read says it: a replay draws its frames at it, so a
// value: duration it draws reads as it did when the world was recorded.
export function recordedAt(recording: Recording): Date | undefined {
    const fleet = recording.exchanges.find(
        (exchange) => exchange.request.watch === 'machines.list' && !('target' in exchange.request),
    );
    const outs = (fleet?.answers ?? []).filter((frame) => isRecord(frame) && 'out' in frame);
    const last = outs.at(-1);
    const out = isRecord(last) ? last.out : undefined;
    const all = isRecord(out) ? out.All : undefined;
    const read = isRecord(all) && typeof all.Read === 'string' ? new Date(all.Read) : undefined;
    return read === undefined || Number.isNaN(read.getTime()) ? undefined : read;
}
