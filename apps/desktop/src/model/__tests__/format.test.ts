import { describe, expect, test } from 'vitest';

import { bytes, duration, memory, percent, rate, seconds } from '../format';

describe('format', () => {
    test('raw bytes read in decimal units, as the engine prints them', () => {
        // Given - sizes the engine gives in bytes
        const sizes = [0, 999, 1500, 5_000_000_000, 18_400_000_000, 994_000_000_000];

        // When - each is shown
        const shown = sizes.map(bytes);

        // Then - each reads in its largest unit, one decimal under 100, as apps/engine/activity/words.go writes it
        expect(shown).toStrictEqual(['0 B', '999 B', '1.5 KB', '5 GB', '18.4 GB', '994 GB']);
    });

    test('a share and a rate read as the engine prints them', () => {
        // Given - two shares and three rates
        // Then - each is in the CLI's units
        expect(percent(0.4)).toBe('0.4');
        expect(percent(34.6)).toBe('35');
        expect(rate(420_000)).toStrictEqual({ unit: 'KB/s', value: '420' });
        expect(rate(12_340_000)).toStrictEqual({ unit: 'MB/s', value: '12.3' });
        expect(rate(1_240_000)).toStrictEqual({ unit: 'MB/s', value: '1.2' });
    });

    test('a size is binary and a span is in its largest unit', () => {
        // Given - two memory sizes and two spans in nanoseconds
        // Then - each reads as the CLI prints it
        expect(memory(13_314_398_618)).toBe('12.4 GB');
        expect(memory(52_428_800)).toBe('50 MB');
        expect(seconds(10_000_000_000)).toBe('10 s');
        expect(seconds(1_800_000_000_000)).toBe('30 min');
    });

    test('a duration leaves out a second unit that reads zero', () => {
        // Given - spans of 4 h, 4 h 10 min, 2 d and 2 d 3 h, in nanoseconds
        const hour = 3_600_000_000_000;
        const spans = [4 * hour, 4 * hour + hour / 6, 48 * hour, 51 * hour];

        // When - each is shown
        const shown = spans.map(duration);

        // Then - "4 h", never "4 h 0 min"
        expect(shown).toStrictEqual(['4 h', '4 h 10 min', '2 d', '2 d 3 h']);
    });
});
