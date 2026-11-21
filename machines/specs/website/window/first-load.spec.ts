import { content, main, region, within } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('first load', () => {
    test('A8 a page not answered yet draws its own panel and says it is reading', async () => {
        // Given - the window on the Passwords page over two-macs, vault.list never answered
        const result = await website.visit(
            '/?world=two-macs&page=passwords&unread=vault.list',
            async (visitor) => {
                // When - the page is drawn
                // Then - it says what it reads, and its drawn panel stays hidden from assistive tech until answered
                await visitor.see(within(main(), content('Reading the passwords of this Mac…')));
                await visitor.gone(region('Passwords'));
            },
        );
        expect(result.errors.text).toBe('');
    });
});
