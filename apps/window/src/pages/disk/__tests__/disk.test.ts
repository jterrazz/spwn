import type { DiskRow } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { byApp, groupsOf, partsOf, rebuildableOf, sizeOf } from '../disk';

function row(App: string, Group: DiskRow['Group'], change: Partial<DiskRow> = {}): DiskRow {
    return { App, Bytes: 1_000_000_000, Group, MoveTo: '', Paths: [], Pending: false, ...change };
}

describe('disk', () => {
    test('rows are grouped in the order the engine first gives each group', () => {
        // Given - rows of three groups, interleaved
        const rows = [
            row('tart', 'misplaced'),
            row('go', 'can-rebuild'),
            row('xcode', 'your-data'),
            row('bun', 'can-rebuild'),
        ];

        // When - they are grouped
        const groups = groupsOf(rows);

        // Then - misplaced, can rebuild, your data, each row in its given order
        expect(
            groups.map((group) => [group.group, group.rows.map((one) => one.App)]),
        ).toStrictEqual([
            ['misplaced', ['tart']],
            ['can-rebuild', ['go', 'bun']],
            ['your-data', ['xcode']],
        ]);
    });

    test("a group's share is the sizes its rows were read at, one part per group", () => {
        // Given - two rows that can be rebuilt, one not read yet, and one misplaced
        const reading = {
            Free: 0,
            Pending: false,
            Rows: [
                row('go', 'can-rebuild', { Bytes: 2_000_000_000 }),
                row('docker', 'can-rebuild', { Bytes: -1 }),
                row('tart', 'misplaced', { Bytes: 7_000_000_000 }),
                row('bun', 'can-rebuild', { Bytes: 500_000_000 }),
            ],
            Total: 0,
        };

        // When - the bar's parts are made
        const parts = partsOf(reading);

        // Then - each group once, in the engine's order, holding the sizes read and no unread one
        expect(parts.map((part) => [part.group, part.bytes])).toStrictEqual([
            ['can-rebuild', 2_500_000_000],
            ['misplaced', 7_000_000_000],
        ]);
    });

    test("an app's rows are one row, but a misplaced folder stays a row of its own", () => {
        // Given - claude's cache and data, go's cache, and tart misplaced twice
        const rows = [
            row('claude', 'can-rebuild'),
            row('tart', 'misplaced', { Paths: ['~/.tart'] }),
            row('go', 'can-rebuild'),
            row('claude', 'your-data'),
            row('tart', 'misplaced', { Paths: ['~/old/.tart'] }),
        ];

        // When - they are gathered by app
        const apps = byApp(rows);

        // Then - claude once with both kinds, each tart folder alone, in the engine's order
        expect(apps.map((app) => [app.app, app.rows.map((one) => one.Group)])).toStrictEqual([
            ['claude', ['can-rebuild', 'your-data']],
            ['tart', ['misplaced']],
            ['go', ['can-rebuild']],
            ['tart', ['misplaced']],
        ]);
    });

    test('what can be rebuilt is the sum of the rebuildable rows read, and nothing else', () => {
        // Given - two rows that can be rebuilt, one unread, and data
        const rows = [
            row('go', 'can-rebuild', { Bytes: 2_000_000_000 }),
            row('docker', 'can-rebuild', { Bytes: -1 }),
            row('xcode', 'your-data', { Bytes: 9_000_000_000 }),
            row('bun', 'can-rebuild', { Bytes: 500_000_000 }),
        ];

        // When - their sum is made
        const held = rebuildableOf(rows);

        // Then - 2.5 GB
        expect(held).toBe(2_500_000_000);
    });

    test('a size the engine has not read is never shown as a number', () => {
        // Given - a row still being read, one past its budget and one read

        // When - their sizes are written
        const sizes = [
            sizeOf(row('go', 'can-rebuild', { Bytes: 0, Pending: true })),
            sizeOf(row('docker', 'can-rebuild', { Bytes: -1 })),
            sizeOf(row('xcode', 'can-rebuild', { Bytes: 5_800_000_000 })),
        ];

        // Then - reading, size not read, then the size
        expect(sizes).toStrictEqual(['reading', 'size not read', '5.8 GB']);
    });
});
