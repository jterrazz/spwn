import { describe, expect, test } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import { StaleMarks } from '../stale-marks';

const REFUSED = new EngineError({
    code: 'remote.unreachable',
    message: 'macbook-air does not answer',
    type: 'unreachable',
});

describe('StaleMarks', () => {
    test('a refusal held by two readers is shown once, and goes when both let it go', () => {
        // Given - two readers holding the one refusal, and a listener
        const marks = new StaleMarks();
        let told = 0;
        marks.subscribe(() => {
            told += 1;
        });
        marks.hold('first', REFUSED);
        marks.hold('second', REFUSED);
        const held = marks.shown();

        // When - both readers let it go
        marks.hold('first', undefined);
        marks.hold('second', undefined);

        // Then - the refusal was shown once, nothing is shown now, and the listener heard each change
        expect(held).toStrictEqual([REFUSED]);
        expect(marks.shown()).toStrictEqual([]);
        expect(told).toBe(4);
    });

    test('holding what a reader already holds tells no one', () => {
        // Given - a reader holding a refusal, and a listener
        const marks = new StaleMarks();
        marks.hold('first', REFUSED);
        let told = 0;
        marks.subscribe(() => {
            told += 1;
        });

        // When - it holds the same refusal again
        marks.hold('first', REFUSED);

        // Then - the listener heard nothing, and the shown list is the one it was
        expect(told).toBe(0);
        expect(marks.shown()).toStrictEqual([REFUSED]);
    });
});
