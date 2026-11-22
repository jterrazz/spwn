import type { DiskAmount, DiskRow } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { DISK_KIND_COLOURS } from '../../../words/tones';
import { byApp, groupsOf, kindPartsOf, listedOf, sizeOf } from '../disk';

function row(App: string, Group: DiskRow['Group'], change: Partial<DiskRow> = {}): DiskRow {
    return { App, Bytes: 1_000_000_000, Group, MoveTo: '', Paths: [], Pending: false, ...change };
}

describe('disk', () => {
    test('every kind of the bar has a colour of its own', () => {
        // Given - the colour each kind of the Disk card takes
        const colours = Object.values(DISK_KIND_COLOURS);

        // Then - no two kinds share one, so the legend never reads two parts as one
        expect(new Set(colours).size).toBe(colours.length);
    });

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

    test('the bar has one part per kind the engine gives, in its order, and none for an absent kind', () => {
        // Given - kinds with no system and no machines, caches before apps
        const kinds: DiskAmount[] = [
            { Bytes: 2_000_000_000, Kind: 'caches' },
            { Bytes: 0, Kind: 'apps' },
            { Bytes: 5_000_000_000, Kind: 'free' },
        ];

        // When - the parts are made
        const parts = kindPartsOf(kinds);

        // Then - those three alone, in that order, the measured zero kept and free as the rest
        expect(parts.map((part) => [part.key, part.bytes, part.rest])).toStrictEqual([
            ['caches', 2_000_000_000, false],
            ['apps', 0, false],
            ['free', 5_000_000_000, true],
        ]);
        expect(kindPartsOf(undefined)).toStrictEqual([]);
    });

    test('a kind or a holder the window does not know is worded by its code and drawn neutral', () => {
        // Given - a kind the window has no colour for
        const unknown = [{ Bytes: 1, Kind: 'swap' }];

        // When - its part is made
        const [part] = kindPartsOf(unknown);

        // Then - its code as its word and the neutral colour
        expect([part?.label, part?.identity]).toStrictEqual(['swap', 'system']);
    });

    test('a part of no bytes is not listed, but free always is, in the engine order', () => {
        // Given - kinds where system and caches hold nothing, apps a byte and free nothing
        const parts = kindPartsOf([
            { Bytes: 0, Kind: 'system' },
            { Bytes: 1, Kind: 'apps' },
            { Bytes: 10_000_000_000, Kind: 'your-data' },
            { Bytes: 0, Kind: 'caches' },
            { Bytes: 0, Kind: 'free' },
        ]);

        // When - the listed parts are kept
        const listed = listedOf(parts);

        // Then - apps and your data, then free though empty
        expect(listed.map((part) => part.key)).toStrictEqual(['apps', 'your-data', 'free']);
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
