import { describe, expect, test } from 'vitest';

import { toolInputSummary, truncateContent } from '../tool-summary';

describe('tool-summary', () => {
    test('each known tool is summarised by the field that tells its calls apart', () => {
        // Given - one call of every tool the summary knows, and one it does not
        const calls = [
            ['Read', { file_path: '/src/a.ts' }],
            ['Bash', { command: `npm test -- ${'x'.repeat(70)}\nsecond line` }],
            ['Grep', { pattern: 'needle' }],
            ['Agent', { description: 'review the change' }],
            ['Unknown', { anything: 'at all' }],
        ] as const;

        // Then - the field, cut to its line and its budget
        expect(calls.map(([tool, input]) => toolInputSummary(tool, input))).toStrictEqual([
            '/src/a.ts',
            `npm test -- ${'x'.repeat(48)}...`,
            'needle',
            'review the change',
            '',
        ]);
    });

    test('a long result keeps its first lines and counts the rest', () => {
        // Given - five lines kept to three
        const content = ['one', 'two', 'three', 'four', 'five'].join('\n');

        // Then - three lines, and how many were left out
        expect(truncateContent(content, 3)).toBe('one\ntwo\nthree\n... (2 more lines)');
    });
});
