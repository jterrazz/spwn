import { button, heading } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('sidebar', () => {
    test.each([
        ['Agents', '/agents'],
        ['Tools', '/tools'],
    ])('the %s entry opens its page', async (entry, path) => {
        // Given - the landing page
        const result = await website.visit('/', async (visitor) => {
            await visitor.see(heading('Worlds'));

            // When - the visitor follows the sidebar entry
            await visitor.click(button(entry));
            await visitor.see(heading(entry));
        });

        // Then - the page under that address is the one shown
        expect(new URL(result.url).pathname).toBe(path);
    });
});
