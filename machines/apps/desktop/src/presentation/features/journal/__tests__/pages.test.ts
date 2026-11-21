import { describe, expect, test } from 'vitest';

import {
    askedOf,
    olderAsked,
    olderOffered,
    pageInput,
    pagingOf,
    saidAfter,
    unheardUnsaid,
} from '../pages';

describe('pages', () => {
    test('asks the first page with no Before, and an older one with its Before, the Mac kept', () => {
        // Given - every Mac, then mac-mini chosen

        // When - the first and the older pages are asked
        const inputs = [
            pageInput(null),
            pageInput(null, '2026-10-29'),
            pageInput('mac-mini'),
            pageInput('mac-mini', '2026-10-29'),
        ];

        // Then - the first reads up to today, the older one before the day Next gave, on the same Mac
        expect(inputs).toStrictEqual([
            {},
            { Before: '2026-10-29' },
            { Mac: 'mac-mini' },
            { Before: '2026-10-29', Mac: 'mac-mini' },
        ]);
    });

    test("offers an older page while the engine's Next names one, and none once it is empty", () => {
        // Given - a page with a Next and the last page

        // When - each is read for the page it offers
        const offered = [olderOffered({ Next: '2026-10-29' }), olderOffered({ Next: '' })];

        // Then - the Before of the next page, then nothing
        expect(offered).toStrictEqual(['2026-10-29', undefined]);
    });

    test('accumulates the older pages asked, oldest last, each once', () => {
        // Given - the History over every Mac, nothing older asked yet
        const first = pagingOf(null);

        // When - two older pages are asked, the second one twice
        const paged = olderAsked(
            olderAsked(olderAsked(first, null, '2026-10-29'), null, '2026-10-15'),
            null,
            '2026-10-15',
        );

        // Then - both, in the order they were asked
        expect(askedOf(first, null)).toStrictEqual([]);
        expect(askedOf(paged, null)).toStrictEqual(['2026-10-29', '2026-10-15']);
    });

    test('starts again from the first page when another Mac is chosen', () => {
        // Given - an older page asked over every Mac
        const paged = olderAsked(pagingOf(null), null, '2026-10-29');

        // When - mac-mini is chosen, then an older page of it asked
        const chosen = askedOf(paged, 'mac-mini');
        const again = olderAsked(paged, 'mac-mini', '2026-10-30');

        // Then - none of every Mac's pages are kept for mac-mini
        expect(chosen).toStrictEqual([]);
        expect(again).toStrictEqual({ asked: ['2026-10-30'], chosen: 'mac-mini' });
    });

    test('names a silent Mac once, on the first page that heard nothing of it', () => {
        // Given - macbook-air unheard on the first page and on the older one, mac-mini on the older one alone
        const air = { Mac: 'macbook-air', Why: 'macbook-air does not answer' };
        const mini = { Mac: 'mac-mini', Why: 'mac-mini does not answer' };
        const said = saidAfter([], { Unheard: [air] });

        // When - the older page's silent Macs are read
        const unsaid = unheardUnsaid(said, { Unheard: [air, mini] });

        // Then - mac-mini alone is said, and both are said after it
        expect(unsaid).toStrictEqual([mini]);
        expect(saidAfter(said, { Unheard: [air, mini] })).toStrictEqual([
            'macbook-air',
            'mac-mini',
        ]);
        expect(unheardUnsaid([], { Unheard: null })).toStrictEqual([]);
    });
});
