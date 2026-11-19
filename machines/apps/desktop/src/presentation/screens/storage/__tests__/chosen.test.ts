import { describe, expect, test } from 'vitest';

import { chosenMac } from '../chosen';

describe('chosen', () => {
    test('this Mac is shown first, and a choice holds while its Mac is listed', () => {
        // Given - two Macs, the second This Mac
        const macs = [
            { Name: 'mini', ThisMac: false },
            { Name: 'book', ThisMac: true },
        ];

        // When - nothing, then mini, then a Mac no longer listed is chosen
        const shown = [
            chosenMac(macs, undefined),
            chosenMac(macs, 'mini'),
            chosenMac(macs, 'gone'),
        ];

        // Then - book, mini, then book again
        expect(shown).toStrictEqual(['book', 'mini', 'book']);
    });
});
