import type { ActivityReport, ActivitySample } from '@jterrazz/station-api';

export const GIB = 1024 ** 3;

const SAMPLE: ActivitySample = {
    at: '2026-11-13T10:14:00Z',
    boot: '2026-11-10T08:00:00Z',
    cpu_busy: 0,
    cpu_total: 0,
    cpus: 8,
    disk_read: 0,
    disk_written: 0,
    load: [],
    mem_total: 16 * GIB,
    mem_used: 8 * GIB,
    net_in: 0,
    net_out: 0,
    swap_used: 0,
    thermal: 0,
};

export function reportPeakingAt(peak: number): ActivityReport {
    return {
        chart: {
            buckets: [
                { at: '2026-11-13T09:59:00Z', max: 40, mean: 20, points: 3 },
                { at: '2026-11-13T10:00:00Z', max: 91, mean: 55, points: 3 },
            ],
            every: 60_000_000_000,
            peak,
            peak_at: '2026-11-13T10:00:00Z',
        },
        findings: null,
        last: {
            sample: SAMPLE,
            top: { cpu: null, memory: null },
            vms: [{ cpu: 12, memory: 2 * GIB, vm: 'linux-1' }],
        },
        rates: { CPU: 23, DiskRead: 0, DiskWrite: 0, NetIn: 0, NetOut: 0 },
        source: 'engine',
        window: 3_600_000_000_000,
    };
}
