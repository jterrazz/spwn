import { content } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { website } from '../website.specification.js';

describe('knowledge', () => {
    test('a world with no knowledge yet points to the architect', async () => {
        // Given - matrix, whose knowledge folder holds nothing
        const result = await website.visit('/world/matrix/knowledge', async (visitor) => {
            await visitor.see(content('No knowledge files yet'));
        });

        // Then - the browser says so and offers the way to start
        expect(result.content.text).toContain(
            'Talk to the Architect to start building your knowledge base',
        );
    });
});
