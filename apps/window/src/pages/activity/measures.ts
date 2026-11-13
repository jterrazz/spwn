import type {
    ActivityBucket,
    ActivityProcess,
    ActivityReading,
    ActivityReport,
    ActivitySample,
    MachinesMacView,
} from '@jterrazz/station-api';

import { clock, memory, percent } from '../../model/format';

export type Span = '1h' | '6m' | '24h';

const NS_PER_SECOND = 1_000_000_000;

export const SPANS: readonly { id: Span; label: string; ns: number }[] = [
    { id: '6m', label: '6 min', ns: 360 * NS_PER_SECOND },
    { id: '1h', label: '1 h', ns: 3600 * NS_PER_SECOND },
    { id: '24h', label: '24 h', ns: 86_400 * NS_PER_SECOND },
];

export function spanOf(id: Span): { id: Span; label: string; ns: number } {
    return SPANS.find((span) => span.id === id) ?? { id, label: id, ns: 0 };
}

export type Marks = { values: readonly (null | number)[]; labels: readonly string[] };

export function marksOf(buckets: readonly ActivityBucket[]): Marks {
    return {
        labels: buckets.map((bucket) =>
            bucket.points > 0
                ? `${clock(bucket.at) ?? bucket.at} · ${percent(bucket.mean)} %`
                : `${clock(bucket.at) ?? bucket.at} · no point`,
        ),
        values: buckets.map((bucket) => (bucket.points > 0 ? bucket.mean : null)),
    };
}

export function cpuOf(report: ActivityReport): string | undefined {
    const { run } = report.last;
    if (run) {
        return percent(run.cpu);
    }
    return report.window > 0 ? percent(report.rates.CPU) : undefined;
}

export type Held = { used: number; total: number | undefined };

export function memoryOf(report: ActivityReport): Held | undefined {
    const { sample, run } = report.last;
    if (run) {
        return { total: undefined, used: run.memory };
    }
    return sample.mem_total > 0 ? { total: sample.mem_total, used: sample.mem_used } : undefined;
}

export function diskOf(sample: ActivitySample): Held | undefined {
    const total = sample.disk_total ?? 0;
    const free = sample.disk_free ?? -1;
    return total > 0 && free >= 0 ? { total, used: total - free } : undefined;
}

export function chartedOf(buckets: readonly ActivityBucket[]): boolean {
    return buckets.some((bucket) => bucket.points > 0);
}

export type Busy = {
    key: string;
    name: string;
    vm: string | undefined;
    share: number;
    shown: string;
};

export function busiestOf(
    processes: readonly ActivityProcess[],
    by: 'cpu' | 'memory',
    total: number,
): readonly Busy[] {
    return processes.map((process) => ({
        key: `${String(process.pid)}/${process.name}`,
        name: process.name,
        share: by === 'cpu' ? process.value : total > 0 ? (process.value / total) * 100 : 0,
        shown: by === 'cpu' ? `${percent(process.value)} %` : memory(process.value),
        vm: process.vm === undefined || process.vm === '' ? undefined : process.vm,
    }));
}

export function busiestWithVms(reading: ActivityReading, by: 'cpu' | 'memory'): readonly Busy[] {
    const total = reading.sample.mem_total;
    const processes = busiestOf(
        (by === 'cpu' ? reading.top.cpu : reading.top.memory) ?? [],
        by,
        total,
    );
    const listed = new Set(processes.map((row) => row.vm));
    const unlisted = busiestOf(
        (reading.vms ?? [])
            .filter((usage) => !listed.has(usage.vm))
            .map((usage) => ({
                name: usage.vm,
                pid: 0,
                value: by === 'cpu' ? usage.cpu : usage.memory,
                vm: usage.vm,
            })),
        by,
        total,
    );
    const vms = [...processes.filter((row) => row.vm !== undefined), ...unlisted];
    return [
        ...vms.toSorted((a, b) => b.share - a.share),
        ...processes.filter((row) => row.vm === undefined),
    ];
}

export function vmOf(mac: MachinesMacView, name: string) {
    return (mac.VMs ?? []).find((vm) => vm.Name === name);
}
