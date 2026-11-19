import { describe, expect, test } from 'vitest';

import { countOf, filtered, numbered } from '../lines';

describe('lines', () => {
    test('a filter keeps the number each line has in the whole log', () => {
        // Given - three lines, filtered on sshd in another case
        const lines = filtered(
            numbered(['kernel: booted', 'sshd[812]: Accepted publickey', 'SSHD[9]: closed']),
            ' sshd ',
        );

        // Then - the two that hold it keep their place
        expect(lines.map((line) => line.number)).toStrictEqual([2, 3]);
    });

    test('the count says what the filter left of the whole', () => {
        // Given - a log of 13 lines, then 2 of them under a filter
        // Then - the whole alone, then the part of the whole
        expect(countOf(13, 13, '')).toBe('13 lines');
        expect(countOf(2, 13, 'sshd')).toBe('2 of 13 lines');
        expect(countOf(1, 1, '')).toBe('1 line');
    });
});
