import { describe, expect, test } from 'vitest';

import { qrOf } from '../qr';

function formatBits(matrix: readonly (readonly boolean[])[]): string {
    const row = matrix[8] ?? [];
    const cells = [0, 1, 2, 3, 4, 5, 7, 8].map((x) => row[x]);
    const column = [7, 5, 4, 3, 2, 1, 0].map((y) => matrix[y]?.[8]);
    return [...cells, ...column].map((dark) => (dark === true ? '1' : '0')).join('');
}

describe('qrOf', () => {
    test('a short address is a version 1 code, its format the low level and mask 0', () => {
        // Given - an address of 17 bytes
        const matrix = qrOf('http://a.b:40012/');

        // Then - 21 modules a side, and the format string the standard gives L with mask 0
        expect(matrix).toHaveLength(21);
        expect(matrix && formatBits(matrix)).toBe('111011111000100');
    });

    test('a tailnet address grows the version, its finders in three corners', () => {
        // Given - a shared address of 52 bytes
        const matrix = qrOf('https://jterrazz-macbook-pro.tail1234.ts.net:40012/');

        // Then - version 3, 29 modules, each finder's centre dark and its ring light
        const corners = [
            [3, 3],
            [25, 3],
            [3, 25],
        ] as const;
        expect(matrix).toHaveLength(29);
        expect(corners.map(([x, y]) => [matrix?.[y]?.[x], matrix?.[y]?.[x + 2]])).toStrictEqual([
            [true, false],
            [true, false],
            [true, false],
        ]);
    });

    test('an address past what version 5 holds has no code', () => {
        // Given - 107 bytes, one past version 5 at the low level
        // Then - 107 bytes is refused, never a truncated code
        expect(qrOf('x'.repeat(107))).toBeUndefined();
    });
});
