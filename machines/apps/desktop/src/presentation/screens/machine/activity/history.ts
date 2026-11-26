import type { ActivityPoint, ActivityRange, ActivityRated } from '@jterrazz/station-api';

import type { Marks } from '../../../../domain/activity/measures';
import { bytes, clock, memory } from '../../../../domain/format';
import { SEPARATOR } from '../../../../domain/separator';

const NS_PER_MS = 1_000_000;

function timeOf(point: ActivityPoint): number {
    return new Date(point.at).getTime();
}

export function keptOver(history: readonly ActivityPoint[], span: number): ActivityPoint[] {
    const last = history.at(-1);
    if (!last) {
        return [];
    }
    const from = timeOf(last) - span / NS_PER_MS;
    return history.filter((point) => timeOf(point) >= from);
}

export function strided<T>(samples: readonly T[], most: number): readonly T[] {
    if (samples.length <= most) {
        return samples;
    }
    return Array.from(
        { length: most },
        (_, index) => samples[Math.floor((index * samples.length) / most)],
    ).filter((sample) => sample !== undefined);
}

export function memoryMarks(points: readonly ActivityPoint[]): Marks {
    return {
        labels: points.map(
            (point) => `${clock(point.at) ?? point.at}${SEPARATOR}${memory(point.memory.mean)}`,
        ),
        values: points.map((point) =>
            point.mem_total > 0 ? (point.memory.mean / point.mem_total) * 100 : null,
        ),
    };
}

// invariant: one mark per run of points, at most `most`: the bytes its points measured, each its mean rate over the
// invariant: time it spans; a run none of whose points measured a rate (a pause, a reboot) has no count, never a zero.
export function countedMarks(
    points: readonly ActivityPoint[],
    most: number,
    rated: (rated: ActivityRated) => ActivityRange,
): Marks {
    const runs = Math.min(points.length, most);
    const steps = Array.from({ length: runs }, (_, index) => {
        const run = points.slice(
            Math.floor((index * points.length) / runs),
            Math.floor(((index + 1) * points.length) / runs),
        );
        const measured = run.flatMap((point) =>
            point.rated === undefined ? [] : [rated(point.rated).mean * point.rated.span],
        );
        return {
            at: run.at(-1)?.at ?? '',
            step: measured.length === 0 ? undefined : measured.reduce((a, b) => a + b, 0),
        };
    });
    const tallest = Math.max(0, ...steps.map(({ step }) => step ?? 0));
    return {
        labels: steps.map(({ at, step }) =>
            step === undefined
                ? `${clock(at) ?? at}${SEPARATOR}no count`
                : `${clock(at) ?? at}${SEPARATOR}${bytes(step)}`,
        ),
        values: steps.map(({ step }) =>
            step === undefined ? null : tallest > 0 ? (step / tallest) * 100 : 0,
        ),
    };
}
