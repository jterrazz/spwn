import { describe, expect, test } from 'vitest';

import { matchesQuery } from '../search';

describe('search', () => {
    test('a query matches a text holding it in any case, and an empty one matches all', () => {
        // Given - the texts of one row
        const texts = ['Visual Studio Code', 'an editor'];

        // When - three queries are asked of it
        const matched = ['  STUDIO ', '', 'terminal'].map((query) => matchesQuery(query, texts));

        // Then - the word in capitals and the empty query match, the absent word does not
        expect(matched).toStrictEqual([true, true, false]);
    });
});
