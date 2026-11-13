import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { GIB, reportPeakingAt } from '../../activity/__tests__/tiles.fixtures';
import { vmCells } from '../home-machines';

describe('home-machines', () => {
    test("a VM's row draws its CPU and its memory from last.vms", async () => {
        // Given - linux-1, 4 GB, which last.vms says uses 12 % and 2 GB
        const report = reportPeakingAt(1);

        // When - its cells are drawn
        const result = await component.render(
            <div>
                {vmCells(
                    { Name: 'linux-1', Resources: { CPUs: 2, DiskBytes: 0, MemoryBytes: 4 * GIB } },
                    report,
                )}
            </div>,
        );

        // Then - both figures are the engine's, and a VM last.vms does not list draws none
        expect(result.html.text).toContain('12 %');
        expect(result.html.text).toContain('2 / 4');
        expect(
            vmCells(
                { Name: 'gate4', Resources: { CPUs: 2, DiskBytes: 0, MemoryBytes: 4 * GIB } },
                report,
            ),
        ).toStrictEqual([null, null]);
    });
});
