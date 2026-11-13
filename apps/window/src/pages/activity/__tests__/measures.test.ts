import { describe, expect, test } from 'vitest';

import { busiestOf, chartedOf, marksOf, spanOf } from '../measures';

describe('measures', () => {
    test('a window is asked in the nanoseconds of a Go duration', () => {
        // Given - the three windows the switch offers
        // Then - each is its span in nanoseconds
        expect(spanOf('6m').ns).toBe(360_000_000_000);
        expect(spanOf('1h').ns).toBe(3_600_000_000_000);
        expect(spanOf('24h').ns).toBe(86_400_000_000_000);
    });

    test('a bucket with no point is a gap, never an idle bar', () => {
        // Given - one bucket measured and one the engine holds no point of
        const marks = marksOf([
            { at: '2026-11-04T10:00:00Z', max: 40, mean: 32, points: 12 },
            { at: '2026-11-04T10:01:00Z', max: 0, mean: 0, points: 0 },
        ]);

        // Then - the first is drawn at its mean, the second is left empty
        expect(marks.values).toStrictEqual([32, null]);
    });

    test('a chart whose every bar holds no point draws nothing', () => {
        // Given - the hour's bars as two-macs answers them, none with a sample
        const empty = [
            { at: '2026-11-04T09:00:00Z', max: 0, mean: 0, points: 0 },
            { at: '2026-11-04T09:01:00Z', max: 0, mean: 0, points: 0 },
        ];

        // Then - it is no chart, and one sampled bar is one
        expect(chartedOf(empty)).toBe(false);
        expect(
            chartedOf([...empty, { at: '2026-11-04T09:02:00Z', max: 9, mean: 4, points: 3 }]),
        ).toBe(true);
    });

    test('a process by memory is drawn against the machine it runs on', () => {
        // Given - a VM's process and a plain one, by memory, on a 16 GiB Mac
        const rows = busiestOf(
            [
                { name: 'Virtualization', pid: 7, value: 8 * 1024 ** 3, vm: 'linux-1' },
                { name: 'WindowServer', pid: 9, value: 1024 ** 3 },
            ],
            'memory',
            16 * 1024 ** 3,
        );

        // Then - each is its share of the whole, the VM named
        expect(rows.map((row) => [row.vm, row.share, row.shown])).toStrictEqual([
            ['linux-1', 50, '8 GB'],
            [undefined, 6.25, '1 GB'],
        ]);
    });
});
