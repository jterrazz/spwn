import { describe, expect, test } from 'vitest';

import { installLinesOf } from '../item-model';

describe('item model', () => {
    test('installs alike on several systems are one line naming each system as sent', () => {
        // Given - node by mise on two systems, a line by another method, and a system sent twice
        const lines = installLinesOf([
            { Fetches: 'node lts', Kinds: ['mac'], Method: 'mise', System: 'macOS' },
            { Fetches: 'node lts', Kinds: ['linux'], Method: 'mise', System: 'Ubuntu' },
            { Fetches: 'node', Kinds: ['macos'], Method: 'brew', System: 'macOS' },
            { Fetches: 'node lts', Kinds: ['macos'], Method: 'mise', System: 'macOS' },
        ]);

        // Then - two lines, in the engine's order, each system once and what is fetched as sent
        expect(lines).toStrictEqual([
            { fetches: 'node lts', method: 'mise', systems: ['macOS', 'Ubuntu'] },
            { fetches: 'node', method: 'brew', systems: ['macOS'] },
        ]);
    });

    test('an install the engine gives no system, fetch or method is a line with those left empty', () => {
        // Given - an install carrying its kinds alone
        const lines = installLinesOf([{ Kinds: ['mac'] }]);

        // Then - nothing is guessed in their place
        expect(lines).toStrictEqual([{ fetches: '', method: '', systems: [] }]);
    });
});
