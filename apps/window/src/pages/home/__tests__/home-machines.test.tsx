import { component } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { GIB, reportPeakingAt } from '../../activity/__tests__/tiles.fixtures';
import { vmCells } from '../home-machines';

const RESOURCES = { CPUs: 2, DiskBytes: 0, MemoryBytes: 4 * GIB };

describe('home-machines', () => {
    test("a VM's row draws what it is for, then its CPU and its memory from last.vms", async () => {
        // Given - linux-1, 4 GB, made for a lot, which last.vms says uses 12 % and 2 GB
        const report = reportPeakingAt(1);
        const purpose = 'Lot L6 · machine file views';

        // When - its cells are drawn
        const result = await component.render(
            <div>
                {vmCells({ Name: 'linux-1', Purpose: purpose, Resources: RESOURCES }, report)}
            </div>,
        );

        // Then - its purpose as the engine keeps it, both figures the engine's, and a VM with neither draws none
        expect(result.html.text).toContain(purpose);
        expect(result.html.text).toContain('12 %');
        expect(result.html.text).toContain('2 / 4');
        expect(vmCells({ Name: 'gate4', Purpose: '', Resources: RESOURCES }, report)).toStrictEqual(
            [null, null, null],
        );
    });
});
