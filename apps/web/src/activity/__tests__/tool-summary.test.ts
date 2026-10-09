import { describe, expect, test } from 'vitest';

import { toolInputSummary, truncateContent } from '../tool-summary';

describe('tool-summary', () => {
    test('each known tool is summarised by the field that tells its calls apart', () => {
        // Given - one call of every tool the summary knows, a pattern at its budget exactly, and one tool it does not know
        const calls = [
            ['Read', { file_path: '/src/a.ts' }],
            ['Edit', { file_path: '/src/b.ts' }],
            ['Write', { file_path: '/src/c.ts\nkept whole' }],
            ['Glob', { pattern: '**/*.ts' }],
            ['Bash', { command: `npm test -- ${'x'.repeat(70)}\nsecond line` }],
            ['Grep', { pattern: 'n'.repeat(40) }],
            ['Agent', { description: 'review the change' }],
            ['Unknown', { anything: 'at all' }],
        ] as const;

        // Then - the field, cut to its line and its budget when the tool has one
        expect(calls.map(([tool, input]) => toolInputSummary(tool, input))).toStrictEqual([
            '/src/a.ts',
            '/src/b.ts',
            '/src/c.ts\nkept whole',
            '**/*.ts',
            `npm test -- ${'x'.repeat(48)}...`,
            'n'.repeat(40),
            'review the change',
            '',
        ]);
    });

    test.each([
        { content: 'one\ntwo\nthree\nfour\nfive', kept: 'one\ntwo\nthree\n... (2 more lines)' },
        { content: 'one\ntwo\nthree', kept: 'one\ntwo\nthree' },
        { content: 'one', kept: 'one' },
    ])(
        'a result of $content.length characters keeps three lines and counts the rest',
        ({ content, kept }) => {
            // Given - a result kept to three lines

            // Then - up to three lines, and how many were left out
            expect(truncateContent(content, 3)).toBe(kept);
        },
    );
});
