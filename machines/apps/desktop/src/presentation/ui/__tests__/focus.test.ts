import { describe, expect, test } from 'vitest';

import { moved, moveOfKey } from '../focus';

describe('focus', () => {
    test.each([
        { expected: 1, index: 0, move: 'next', wrap: true },
        { expected: 0, index: 2, move: 'next', wrap: true },
        { expected: 2, index: 0, move: 'previous', wrap: true },
        { expected: 2, index: 2, move: 'next', wrap: false },
        { expected: 0, index: 0, move: 'previous', wrap: false },
        { expected: 0, index: -1, move: 'next', wrap: false },
        { expected: 2, index: -1, move: 'previous', wrap: true },
        { expected: 0, index: 1, move: 'first', wrap: false },
        { expected: 2, index: 1, move: 'last', wrap: false },
    ] as const)('$move from $index of three, wrapping $wrap, lands on $expected', (step) => {
        // Given - three controls, the keyboard on one of them or outside them (-1)
        const index = moved(3, step.index, step.move, step.wrap);

        // Then - a trap wraps around its ends, a list stops at them
        expect(index).toBe(step.expected);
    });

    test('nothing to move to is no index', () => {
        // Given - a dialog that holds no control
        // Then - the move lands nowhere
        expect(moved(0, -1, 'next', true)).toBe(-1);
    });

    test.each([
        { axis: 'vertical', expected: 'next', key: 'ArrowDown' },
        { axis: 'vertical', expected: 'previous', key: 'ArrowUp' },
        { axis: 'vertical', expected: undefined, key: 'ArrowRight' },
        { axis: 'horizontal', expected: 'next', key: 'ArrowRight' },
        { axis: 'horizontal', expected: 'previous', key: 'ArrowLeft' },
        { axis: 'horizontal', expected: 'last', key: 'End' },
        { axis: 'vertical', expected: 'first', key: 'Home' },
        { axis: 'vertical', expected: undefined, key: 'a' },
    ] as const)('$key on a $axis list moves to $expected', (step) => {
        // Given - a key pressed on a list of that axis
        // When - its move is read
        const move = moveOfKey(step.key, step.axis);

        // Then - it is the move of that axis, or none
        expect(move).toBe(step.expected);
    });
});
