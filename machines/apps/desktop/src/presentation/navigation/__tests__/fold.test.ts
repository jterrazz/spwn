import { describe, expect, test } from 'vitest';

import { BAR_FIXED, fixedWidth, FOLD_WIDTH, foldOf } from '../fold';

describe('fold', () => {
    test('a path that fits folds nothing', () => {
        // Given - Machine, a Mac and a VM in a wide bar
        const folded = foldOf([130, 140, 110], 600);

        // Then - every segment shows
        expect(folded).toStrictEqual([]);
    });

    test('a path too long folds the levels between the first and the last, from the first onward', () => {
        // Given - four segments in a room for about three
        const folded = foldOf([130, 200, 200, 120], 500);

        // Then - the second folds, and the third with it only if the `…` still leaves no room
        expect(folded).toStrictEqual([1]);
        expect(foldOf([130, 200, 200, 120], 300)).toStrictEqual([1, 2]);
    });

    test('the first and the last never fold, however narrow the room', () => {
        // Given - two segments in no room at all
        const folded = foldOf([130, 400], 10);

        // Then - nothing folds
        expect(folded).toStrictEqual([]);
    });

    test('at the window minimum, the parts that never fold leave room for the space, the `…` and a last segment', () => {
        // Given - the 1100 px minimum, the column hidden, and a last segment of 160 px
        const least = fixedWidth() + 140 + FOLD_WIDTH + 160;

        // Then - the bar holds them, the Find field kept at its 120 px
        expect(BAR_FIXED.find).toBe(120);
        expect(least).toBeLessThanOrEqual(1100);
    });
});
