import { describe, expect, test } from 'vitest';

import { categoriesOf, countIn, EVERY_CATEGORY, matches } from '../model';
import type { Package } from '../model';

const gh: Package = { category: 'Git', description: 'GitHub CLI', name: 'gh' };
const node: Package = { category: 'Runtimes', description: 'JavaScript runtime', name: 'node' };
const lazygit: Package = { category: 'Git', description: 'A terminal UI for git', name: 'lazygit' };

describe('model', () => {
    test('the categories are the ones the packages carry, in the order the engine lists them', () => {
        // Given - packages whose categories the engine gave
        const categories = categoriesOf([gh, node, lazygit]);

        // Then - each category once, first seen first
        expect(categories).toStrictEqual(['Git', 'Runtimes']);
    });

    test('a category pill counts the packages in it', () => {
        // Given - two Git packages and a runtime
        const count = countIn([gh, node, lazygit], 'Git');

        // Then - the pill says two
        expect(count).toBe(2);
    });

    test('the search reads the name, the description and the category, in any case', () => {
        // Given - a search for a word of node's description
        const found = [gh, node, lazygit].filter((item) =>
            matches(item, EVERY_CATEGORY, 'JAVASCRIPT'),
        );

        // Then - node alone matches
        expect(found).toStrictEqual([node]);
    });

    test('a category chosen keeps only its packages, and the search narrows it further', () => {
        // Given - the Git category chosen, then a search for "terminal"
        const inGit = [gh, node, lazygit].filter((item) => matches(item, 'Git', ''));
        const searched = inGit.filter((item) => matches(item, 'Git', 'terminal'));

        // Then - both Git packages, then lazygit alone
        expect(inGit).toStrictEqual([gh, lazygit]);
        expect(searched).toStrictEqual([lazygit]);
    });
});
