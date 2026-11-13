import type { CatalogLivesOn } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { installLinesOf, targetOf } from '../item-model';

const mise: CatalogLivesOn['Install'] = { App: 'node', Version: 'lts' };

describe('item model', () => {
    test("an install's target is its text values, as sent, in the order sent", () => {
        // Given - the shapes three methods send
        const script = targetOf({ Args: null, Check: '', URL: 'https://claude.ai/install.sh' });
        const brew = targetOf({ Formula: 'cocoapods', Trusted: false });
        const apt = targetOf({ Packages: ['git', 'curl'] });

        // Then - each its text, the empty and the non-text values left out
        expect(targetOf(mise)).toBe('node · lts');
        expect(script).toBe('https://claude.ai/install.sh');
        expect(brew).toBe('cocoapods');
        expect(apt).toBe('git curl');
    });

    test('installs alike on several systems are one line naming each system', () => {
        // Given - node by mise on three systems, and a mac-only line by another method
        const lines = installLinesOf([
            { Install: mise, Kinds: ['mac'], Method: 'mise' },
            { Install: mise, Kinds: ['linux'], Method: 'mise' },
            { Install: { Formula: 'node' }, Kinds: ['macos'], Method: 'brew' },
            { Install: mise, Kinds: ['macos', 'mac'], Method: 'mise' },
        ]);

        // Then - two lines, in the engine's order, each system once
        expect(lines).toStrictEqual([
            { method: 'mise', systems: ['mac', 'linux', 'macos'], target: 'node · lts' },
            { method: 'brew', systems: ['macos'], target: 'node' },
        ]);
    });
});
