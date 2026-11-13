import { component, region } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { ENTRIES } from '../entries';
import Gallery from '../gallery';

const design: object = await import('../../../design/index');

// value: what design exports that draws nothing of its own: Screen is a scope a page's one primary is held in.
const DRAWS_NOTHING = new Set(['Screen']);

describe('Gallery', () => {
    test('every component design exports has its entry, and each entry draws its stories', () => {
        // Given - every component the design layer exports
        const components = Object.keys(design).filter(
            (name) => /^[A-Z][a-z]/u.test(name) && !DRAWS_NOTHING.has(name),
        );

        // When - the gallery's entries are read
        const entered = new Set(ENTRIES.map((one) => one.name));

        // Then - none is left out, and none is entered without a story
        expect(components.filter((name) => !entered.has(name))).toStrictEqual([]);
        expect(ENTRIES.filter((one) => one.samples.length === 0)).toStrictEqual([]);
    });

    test('the gallery draws every entry, grouped by layer, and nothing fails', async () => {
        // Given - the gallery page
        const result = await component.render(<Gallery />, async (visitor) => {
            // When - it has drawn its last layer
            await visitor.see(region('Dialogs'));
        });

        // Then - one section per entry, and no error on the way
        expect(result.html.text.match(/data-entry="/gu)).toHaveLength(ENTRIES.length);
        expect(result.errors.text).toBe('');
    });
});
