import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('providers', () => {
    test('a host with no credentials lists every provider as not configured', async () => {
        // Given - a spwn home with no keychain and no provider token
        const result = await website.visit('/providers', async (visitor) => {
            await visitor.see(heading('Settings'));
            await visitor.see(button('How to connect', { exact: false }));
        });

        // Then - each provider offers to reconnect and explains how
        expect(result.content.text).toContain('Not configured');
    });
});
