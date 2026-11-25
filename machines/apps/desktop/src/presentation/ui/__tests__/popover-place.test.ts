import { describe, expect, test } from 'vitest';

import { placed } from '../popover-place';

const WINDOW = { height: 720, left: 0, top: 0, width: 1280 };

const POPOVER = { height: 400, width: 304 };

describe('placed', () => {
    test.each([
        {
            align: 'start',
            anchor: { height: 32, left: 330, top: 10, width: 100 },
            expected: { height: 400, side: 'below', x: 330, y: 48 },
            name: 'room below: under its opener, at its start',
            size: POPOVER,
        },
        {
            align: 'end',
            anchor: { height: 32, left: 900, top: 10, width: 100 },
            expected: { height: 400, side: 'below', x: 696, y: 48 },
            name: 'aligned at its end: its right edge on the opener',
            size: POPOVER,
        },
        {
            align: 'start',
            anchor: { height: 32, left: 1200, top: 10, width: 60 },
            expected: { height: 400, side: 'below', x: 968, y: 48 },
            name: 'past the right edge: shifted back inside the margin',
            size: POPOVER,
        },
        {
            align: 'end',
            anchor: { height: 32, left: 2, top: 10, width: 60 },
            expected: { height: 400, side: 'below', x: 8, y: 48 },
            name: 'past the left edge: shifted back inside the margin',
            size: POPOVER,
        },
        {
            align: 'start',
            anchor: { height: 32, left: 330, top: 600, width: 100 },
            expected: { height: 400, side: 'above', x: 330, y: 194 },
            name: 'cut below and more room above: above its opener',
            size: POPOVER,
        },
        {
            align: 'start',
            anchor: { height: 32, left: 330, top: 300, width: 100 },
            expected: { height: 374, side: 'below', x: 330, y: 338 },
            name: 'taller than either side: the roomier side, its height capped',
            size: { height: 1000, width: 304 },
        },
    ] as const)('$name', ({ anchor, size, align, expected }) => {
        // Given - an opener in a 1280 x 720 window
        // When - its popover is placed
        const place = placed(anchor, size, WINDOW, align);

        // Then - it lands where the window holds it whole, or as much of it as fits
        expect(place).toStrictEqual(expected);
    });

    test('a frame inside the page places in its own coordinates', () => {
        // Given - a frame at (100, 50), an opener near its top-left corner
        const frame = { height: 480, left: 100, top: 50, width: 600 };

        // When - the popover is placed in that frame
        const place = placed(
            { height: 30, left: 120, top: 70, width: 80 },
            POPOVER,
            frame,
            'start',
        );

        // Then - its position is counted from the frame's corner
        expect(place).toStrictEqual({ height: 400, side: 'below', x: 20, y: 56 });
    });
});
