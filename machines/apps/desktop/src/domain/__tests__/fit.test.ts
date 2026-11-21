import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { fitOf, PAGE_NEWEST, PAGE_OLDEST } from '../fit';

const ENGINE_PAGE = join(import.meta.dirname, '..', '..', '..', '..', 'engine', 'page', 'page.go');

describe('fit', () => {
    test('the verdict is a range from the oldest accepted to the newest known', () => {
        // Given - a window that accepts revisions 3 to 5, and engines naming none, 2, 3 to 5 and 6

        // When - each is judged
        const verdicts = [undefined, 2, 3, 4, 5, 6].map((revision) =>
            fitOf(revision, { newest: 5, oldest: 3 }),
        );

        // Then - below the oldest or none named is an older engine, above the newest an older window, between fits
        expect(verdicts).toStrictEqual([
            'engine-older',
            'engine-older',
            'fits',
            'fits',
            'fits',
            'app-older',
        ]);
    });

    test("the window's range holds its tree's engine, whose revision is the newest it knows", () => {
        // Given - the revision the engine of this tree speaks
        const revision = /const Revision = (?<n>\d+)/u.exec(readFileSync(ENGINE_PAGE, 'utf8'))
            ?.groups?.n;

        // When - the window's own range is read against it
        const range = { newest: PAGE_NEWEST, oldest: PAGE_OLDEST };

        // Then - the window knows that revision as its newest, and accepts no newer than it knows
        expect(Number(revision)).toBe(range.newest);
        expect(range.oldest).toBeLessThanOrEqual(range.newest);
        expect(fitOf(range.newest)).toBe('fits');
    });
});
