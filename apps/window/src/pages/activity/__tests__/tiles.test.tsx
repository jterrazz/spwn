import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cpuFootOf, CpuTile } from '../tiles';
import { reportPeakingAt } from './tiles.fixtures';

describe('tiles', () => {
    test("the CPU tile's foot is the busiest bucket's max, and its age", async () => {
        // Given - an hour's chart whose second bucket holds the busiest point, 14 min before the sample
        const report = reportPeakingAt(1);

        // When - the tile is drawn
        const result = await component.render(
            <CpuTile label="CPU, this Mac" name="jterrazz-macbook-pro" report={report} />,
        );

        // Then - it says the bucket's max, never the bucket's index, and how long ago
        expect(cpuFootOf(report)).toBe('Peak 91 % · 14 min ago');
        expect(result.tree.text).toContain('Peak 91 % · 14 min ago');
    });

    test('a chart whose no bucket holds a point falls back to the CPU count', () => {
        // Given - the same answer, its peak -1 as the engine sends it with no point
        const report = reportPeakingAt(-1);

        // Then - the foot counts the CPUs
        expect(cpuFootOf(report)).toBe('8 CPUs');
    });
});
