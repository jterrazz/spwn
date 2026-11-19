import { describe, expect, test } from 'vitest';

import { dayAndClock } from '../when';

describe('when', () => {
    test('a moment reads as its day, a comma and its 24 h clock', () => {
        // Given - 6 Nov at 09:12 and 21:05 of the local day
        const morning = new Date(2026, 10, 6, 9, 12).toISOString();
        const evening = new Date(2026, 10, 6, 21, 5).toISOString();

        // Then - each reads day, short month, comma, two-digit hours and minutes
        expect(dayAndClock(morning)).toBe('6 Nov, 09:12');
        expect(dayAndClock(evening)).toBe('6 Nov, 21:05');
    });

    test('an absent or zero moment reads as nothing', () => {
        // Given - a field the wire left out, an empty one and the zero date
        // Then - no word and no zero date stands in
        expect(dayAndClock(undefined)).toBeUndefined();
        expect(dayAndClock('')).toBeUndefined();
        expect(dayAndClock('0001-01-01T00:00:00Z')).toBeUndefined();
    });
});
