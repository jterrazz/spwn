import { describe, expect, test } from 'vitest';

import { pinnedOf } from '../pin';

describe('pin', () => {
    test.each([
        { chosen: undefined, expected: true, width: 1440 },
        { chosen: undefined, expected: true, width: 1920 },
        { chosen: undefined, expected: false, width: 1439 },
        { chosen: undefined, expected: false, width: 1100 },
        { chosen: 'pinned', expected: true, width: 1100 },
        { chosen: 'hidden', expected: false, width: 1920 },
    ] as const)('$chosen at $width px is pinned: $expected', ({ chosen, width, expected }) => {
        // Given - the choice the window keeps, if any, and its width
        const pinned = pinnedOf(chosen, width);

        // Then - a choice wins, and without one the width decides from 1440 px
        expect(pinned).toBe(expected);
    });
});
