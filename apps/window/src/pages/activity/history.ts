import type { ActivitySample } from '@jterrazz/station-api';

import { SEPARATOR } from '../../design/separator';
import { bytes, clock, memory } from '../../model/format';
import type { Marks } from './measures';

const NS_PER_MS = 1_000_000;

function timeOf(sample: ActivitySample): number {
    return new Date(sample.at).getTime();
}

export function keptOver(history: readonly ActivitySample[], span: number): ActivitySample[] {
    const last = history.at(-1);
    if (!last) {
        return [];
    }
    const from = timeOf(last) - span / NS_PER_MS;
    return history.filter((sample) => timeOf(sample) >= from);
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

export function memoryMarks(samples: readonly ActivitySample[]): Marks {
    return {
        labels: samples.map(
            (sample) => `${clock(sample.at) ?? sample.at}${SEPARATOR}${memory(sample.mem_used)}`,
        ),
        values: samples.map((sample) =>
            sample.mem_total > 0 ? (sample.mem_used / sample.mem_total) * 100 : null,
        ),
    };
}

export function countedMarks(
    samples: readonly ActivitySample[],
    counted: (sample: ActivitySample) => number,
): Marks {
    const steps = samples.slice(1).map((sample, index) => {
        const before = samples[index];
        const step = before === undefined ? -1 : counted(sample) - counted(before);
        return { at: sample.at, step: step >= 0 ? step : undefined };
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
