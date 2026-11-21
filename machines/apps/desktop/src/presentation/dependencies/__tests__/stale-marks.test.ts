import { describe, expect, test } from 'vitest';

import { EngineError } from '../../../domain/engine-error';
import { StaleMarks } from '../stale-marks';

const REFUSED = new EngineError({
    code: 'remote.unreachable',
    message: 'macbook-air does not answer',
    type: 'unreachable',
});

const EARLIER = new Date('2026-11-13T10:42:00Z');
const LATER = new Date('2026-11-13T10:43:00Z');

describe('StaleMarks', () => {
    test('a refusal held by two readers is shown once, as of the older answer, and goes when both let it go', () => {
        // Given - two readers holding the one refusal over answers of two instants, and a listener
        const marks = new StaleMarks();
        let told = 0;
        marks.subscribe(() => {
            told += 1;
        });
        marks.hold('first', { at: LATER, error: REFUSED });
        marks.hold('second', { at: EARLIER, error: REFUSED });
        const held = marks.shown();

        // When - both readers let it go
        marks.hold('first', undefined);
        marks.hold('second', undefined);

        // Then - the refusal was shown once with the older instant, nothing is shown now, and each change was heard
        expect(held).toStrictEqual([{ at: EARLIER, error: REFUSED }]);
        expect(marks.shown()).toStrictEqual([]);
        expect(told).toBe(4);
    });

    test('holding what a reader already holds tells no one', () => {
        // Given - a reader holding a refusal, and a listener
        const marks = new StaleMarks();
        marks.hold('first', { at: EARLIER, error: REFUSED });
        let told = 0;
        marks.subscribe(() => {
            told += 1;
        });

        // When - it holds the same refusal over the same instant again, as new objects
        marks.hold('first', { at: new Date(EARLIER), error: REFUSED });

        // Then - the listener heard nothing, and the shown list is the one it was
        expect(told).toBe(0);
        expect(marks.shown()).toStrictEqual([{ at: EARLIER, error: REFUSED }]);
    });
});
