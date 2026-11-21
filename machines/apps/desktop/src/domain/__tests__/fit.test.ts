import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { fitOf, PAGE_REVISION, STATE_REVISION } from '../fit';

const ENGINE_PAGE = join(import.meta.dirname, '..', '..', '..', '..', 'engine', 'page', 'page.go');
const ENGINE_STATE = join(
    import.meta.dirname,
    '..',
    '..',
    '..',
    '..',
    'engine',
    'engine',
    'state.go',
);

describe('fit', () => {
    test('only the revision the window draws fits', () => {
        // Given - a window that draws revision 5, and engines naming none, 3, 4, 5 and 6

        // When - each is judged
        const verdicts = [undefined, 3, 4, 5, 6].map((revision) => fitOf(revision, 5));

        // Then - below it or none named is an older engine, above it an older window, and only 5 fits
        expect(verdicts).toStrictEqual([
            'engine-older',
            'engine-older',
            'engine-older',
            'fits',
            'app-older',
        ]);
    });

    test("the window draws its tree's engine's revision", () => {
        // Given - the revision the engine of this tree speaks
        const revision = /const Revision = (?<n>\d+)/u.exec(readFileSync(ENGINE_PAGE, 'utf8'))
            ?.groups?.n;

        // When - the window's own revision is read against it

        // Then - the window draws that revision, and it fits
        expect(Number(revision)).toBe(PAGE_REVISION);
        expect(fitOf(PAGE_REVISION)).toBe('fits');
    });

    test("the window reads its tree's engine's state revision", () => {
        // Given - the revision of station.state/v1 the engine of this tree speaks
        const revision = /StateRevision = (?<n>\d+)/u.exec(readFileSync(ENGINE_STATE, 'utf8'))
            ?.groups?.n;

        // When - the window's own revision is read against it

        // Then - the window reads that revision
        expect(Number(revision)).toBe(STATE_REVISION);
    });
});
