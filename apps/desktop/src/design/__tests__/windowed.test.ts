import { describe, expect, test } from 'vitest';

import { heightOf, spanOf } from '../windowed';

const AT = { count: 5000, height: 420, overscan: 10, row: 21 };

describe('windowed', () => {
    test('the top of a long list draws its first lines and an overscan', () => {
        // Given - 5 000 lines of 21 px in a box of 420 px, scrolled to the top
        const span = spanOf({ ...AT, top: 0 });

        // Then - twenty lines are seen, ten more are drawn under them
        expect(span).toStrictEqual({ end: 30, start: 0 });
    });

    test('a list scrolled deep draws the lines around its view alone', () => {
        // Given - the same list scrolled to line 4 000
        const span = spanOf({ ...AT, top: 4000 * 21 });

        // Then - the window holds the lines of the view and ten on each side
        expect(span).toStrictEqual({ end: 4030, start: 3990 });
    });

    test('the end of the list is drawn whole, never past it', () => {
        // Given - the list scrolled past its end
        const span = spanOf({ ...AT, top: 6000 * 21 });

        // Then - the window stops at the last line
        expect(span.end).toBe(5000);
        expect(span.start).toBeLessThan(5000);
    });

    test('no line is an empty window', () => {
        // Given - a filter that left nothing
        // Then - nothing is drawn
        expect(spanOf({ ...AT, count: 0, top: 0 })).toStrictEqual({ end: 0, start: 0 });
    });

    test('the box is as tall as its lines, up to its most', () => {
        // Given - three lines, then 5 000
        // Then - three lines take their height, 5 000 the most the box holds
        expect(heightOf(3, 21, 546)).toBe(63);
        expect(heightOf(5000, 21, 546)).toBe(546);
    });
});
