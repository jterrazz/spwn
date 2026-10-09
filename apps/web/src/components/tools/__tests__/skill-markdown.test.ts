import { describe, expect, test } from 'vitest';

import { inlineRuns, skillBlocks } from '../skill-markdown';

describe('skill-markdown', () => {
    test('a skill file reads as its headings, items, code and paragraphs, its title left out', () => {
        // Given - a skill file using every form the page renders
        const content = [
            '# Codex',
            '',
            'Codex runs tasks.',
            '## Usage',
            '```bash',
            'codex "task"',
            '```',
            '- one',
            '2. two',
            '### Notes',
        ].join('\n');

        // Then - one block per form, in order
        expect(skillBlocks(content)).toStrictEqual([
            { kind: 'paragraph', text: 'Codex runs tasks.' },
            { kind: 'heading', level: 2, text: 'Usage' },
            { kind: 'code', text: 'codex "task"' },
            { kind: 'item', marker: '-', text: 'one' },
            { kind: 'item', marker: '2.', text: 'two' },
            { kind: 'heading', level: 3, text: 'Notes' },
        ]);
    });

    test('a line splits into plain runs and code spans', () => {
        // Given - a line with two code spans
        const line = 'run `npm test` then `make`';

        // Then - the runs, each marked
        expect(inlineRuns(line)).toStrictEqual([
            { code: false, text: 'run ' },
            { code: true, text: 'npm test' },
            { code: false, text: ' then ' },
            { code: true, text: 'make' },
        ]);
    });
});
