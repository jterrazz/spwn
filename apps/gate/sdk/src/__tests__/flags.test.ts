import { describe, expect, test } from 'vitest';

import { parseFlags } from '../flags.ts';

describe('parseFlags', () => {
    test.each([
        { argv: [], flags: {} },
        { argv: ['--text', 'hello'], flags: { text: 'hello' } },
        { argv: ['--dry-run'], flags: { 'dry-run': true } },
        { argv: ['--text', 'hello', '--dry-run'], flags: { 'dry-run': true, text: 'hello' } },
        { argv: ['--dry-run', '--text', 'hello'], flags: { 'dry-run': true, text: 'hello' } },
        { argv: ['stray', '--limit', '5', 'other'], flags: { limit: '5' } },
    ])('reads $argv as $flags', ({ argv, flags }) => {
        // Given - the arguments after the method name

        // Then - a value follows its key, a key followed by another key or nothing is a bare flag, anything else is skipped
        expect(parseFlags(argv)).toStrictEqual(flags);
    });
});
