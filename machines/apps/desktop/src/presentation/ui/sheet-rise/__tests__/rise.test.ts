import { describe, expect, test } from 'vitest';

import { PullGate } from '../pull-gate';
import { dragVerdict, RISE, WheelGate } from '../rise';

describe('rise', () => {
    test('a wheel over the strip raises past 30 px summed, its gaps under 220 ms', () => {
        // Given - the strip's gate
        const gate = new WheelGate(RISE.wheel.strip);

        // When - 20 px, then 15 px 200 ms later; then a fresh gesture of 20 px after a 300 ms gap
        const told = [gate.push(20, 0), gate.push(15, 200), gate.push(20, 500)];

        // Then - the second step crosses 30, the third starts again from zero
        expect(told).toStrictEqual([false, true, false]);
    });

    test('a wheel past the canvas bottom raises past 90 px, a gap over 220 ms starting it again', () => {
        // Given - the canvas's gate
        const gate = new WheelGate(RISE.wheel.canvas);

        // When - 60 then 60 px 300 ms apart, then 40 px 100 ms later
        const told = [gate.push(60, 0), gate.push(60, 300), gate.push(40, 400)];

        // Then - only the sum within one gesture counts
        expect(told).toStrictEqual([false, false, true]);
    });

    test('a pull at the page top lowers past 110 px, never in a gesture that scrolled', () => {
        // Given - two pulls, one that scrolled first
        const still = new PullGate();
        const scrolled = new PullGate();

        // When - each wheels up 60 px twice
        const fromTop = [still.push(-60, 0, false), still.push(-60, 100, false)];
        const afterScroll = [
            scrolled.push(-60, 0, true),
            scrolled.push(-60, 100, false),
            scrolled.push(-60, 200, false),
        ];

        // Then - only the one that stayed at the top lowers
        expect(fromTop).toStrictEqual([false, true]);
        expect(afterScroll).toStrictEqual([false, false, false]);
    });

    test.each([
        { moved: -61, raised: false, took: 1000, verdict: 'raise' },
        { moved: -59, raised: false, took: 1000, verdict: 'stay' },
        { moved: -30, raised: false, took: 50, verdict: 'raise' },
        { moved: 91, raised: true, took: 1000, verdict: 'lower' },
        { moved: 89, raised: true, took: 1000, verdict: 'stay' },
        { moved: 40, raised: true, took: 50, verdict: 'lower' },
    ] as const)(
        'a drag of $moved px over $took ms, raised $raised, lands $verdict',
        ({ moved, raised, took, verdict }) => {
            // Given - a drag released after moving `moved` px over `took` ms
            const landed = dragVerdict(raised, moved, took);

            // Then - the prototype's distances and speeds decide
            expect(landed).toBe(verdict);
        },
    );
});
