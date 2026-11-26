import type { DiskAmount, DiskRow } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { DISK_KIND_STEPS } from '../../../words/tones';
import { byApp, groupsOf, isLow, kindPartsOf, listedOf, sizeOf } from '../disk';

function row(App: string, Group: DiskRow['Group'], change: Partial<DiskRow> = {}): DiskRow {
    return { App, Bytes: 1_000_000_000, Group, MoveTo: '', Paths: [], Pending: false, ...change };
}

describe('disk', () => {
    test('a kind that holds bytes is drawn in a neutral step, never a hue, and free in none', () => {
        // Given - the step each kind of the Disk card takes, and the kinds the engine can give
        const steps = Object.values(DISK_KIND_STEPS);
        const parts = kindPartsOf(
            [...Object.keys(DISK_KIND_STEPS), 'free'].map((Kind) => ({ Bytes: 1, Kind })),
        );

        // Then - each step is one of the four greys, and the one part left over, free, has none
        expect(new Set(steps)).toStrictEqual(new Set(['t2', 't3', 't4', 's3']));
        expect(
            parts.filter((part) => part.step === undefined).map((part) => part.key),
        ).toStrictEqual(['free']);
    });

    test('a disk is low under a tenth free, the engine line, and never merely for being used', () => {
        // Given - a disk 4 % free, one exactly a tenth free, one 9 % free, one unread and one of no size
        const verdicts = [
            isLow({ Free: 4, Total: 100 }),
            isLow({ Free: 10, Total: 100 }),
            isLow({ Free: 9, Total: 100 }),
            isLow({ Free: -1, Total: 100 }),
            isLow({ Free: 0, Total: 0 }),
        ];

        // Then - only the two under a tenth are low
        expect(verdicts).toStrictEqual([true, false, true, false, false]);
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
        // Given - a kind the window has no step for
        const unknown = [{ Bytes: 1, Kind: 'swap' }];

        // When - its part is made
        const [part] = kindPartsOf(unknown);

        // Then - its code as its word and the middle grey
        expect([part?.label, part?.step]).toStrictEqual(['swap', 't3']);
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
