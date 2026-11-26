import { describe, expect, test } from 'vitest';

import { Batch } from '../batch';

// value: a store of one listener set, told by hand, as the read store and the mirror tell theirs.
function source() {
    const listeners = new Set<() => void>();
    return {
        subscribe: (listener: () => void) => {
            listeners.add(listener);
            return () => {
                listeners.delete(listener);
            };
        },
        tell: () => {
            for (const listener of listeners) {
                listener();
            }
        },
    };
}

// value: a schedule the spec runs by hand, so a turn is a step of the test and never a timer.
function turns() {
    const planned: (() => void)[] = [];
    return {
        next: () => {
            for (const flush of planned.splice(0)) {
                flush();
            }
        },
        planned,
        schedule: (flush: () => void) => {
            planned.push(flush);
        },
    };
}

describe('Batch', () => {
    test('listeners of two sources told in one turn are told once each, together, at the next', () => {
        // Given - two sources, a listener on each through one batch, the first source told twice
        const turn = turns();
        const batch = new Batch(turn.schedule);
        const [first, second] = [source(), source()];
        const told: string[] = [];
        batch.of(first.subscribe)(() => {
            told.push('first');
        });
        batch.of(second.subscribe)(() => {
            told.push('second');
        });
        first.tell();
        second.tell();
        first.tell();
        const before = [...told];

        // When - the next turn comes
        turn.next();

        // Then - nothing was told in the turn itself, then each listener once, in one flush planned once
        expect(before).toStrictEqual([]);
        expect(told).toStrictEqual(['first', 'second']);
        expect(turn.planned).toStrictEqual([]);
    });

    test('a listener let go of before its turn is not told', () => {
        // Given - a listener on a source through a batch, the source told
        const turn = turns();
        const batch = new Batch(turn.schedule);
        const one = source();
        let told = 0;
        const leave = batch.of(one.subscribe)(() => {
            told += 1;
        });
        one.tell();

        // When - the listener lets go, then the turn comes
        leave();
        turn.next();

        // Then - it was never told
        expect(told).toBe(0);
    });

    test('a source is batched as the same function each time, so a subscription is kept', () => {
        // Given - a batch and a source
        const batch = new Batch(turns().schedule);
        const one = source();

        // When - the source is batched twice
        const twice = [batch.of(one.subscribe), batch.of(one.subscribe)];

        // Then - both are the one function
        expect(twice[0]).toBe(twice[1]);
    });
});
