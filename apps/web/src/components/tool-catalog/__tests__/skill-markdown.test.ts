import { describe, expect, test } from 'vitest';

import { inlineRuns, skillBlocks } from '../skill-markdown';

describe('skill-markdown', () => {
    test('a skill file reads as its headings, items, code and paragraphs, its title left out', () => {
        // Given - a skill file using every form the page renders, indented, blank and unclosed ones included
        const content = [
            '# Codex',
            '',
            'Codex runs tasks.',
            '## Usage',
            '```bash',
            'codex "task"',
            'codex --help',
            '```',
            '- one',
            '  - nested',
            '12.  twelve',
            '   ',
            '### Notes',
            '  ```',
            'first',
            'second',
        ].join('\n');

        // Then - one block per form, in order
        expect(skillBlocks(content)).toStrictEqual([
            { kind: 'paragraph', text: 'Codex runs tasks.' },
            { kind: 'heading', level: 2, text: 'Usage' },
            { kind: 'code', text: 'codex "task"\ncodex --help' },
            { kind: 'item', marker: '-', text: 'one' },
            { kind: 'item', marker: '-', text: 'nested' },
            { kind: 'item', marker: '12.', text: 'twelve' },
            { kind: 'heading', level: 3, text: 'Notes' },
            { kind: 'code', text: 'first\nsecond' },
        ]);
    });

    test('a file closed outside any fence ends on its last block', () => {
        // Given - a single paragraph
        // Then - that paragraph alone
        expect(skillBlocks('Just text')).toStrictEqual([{ kind: 'paragraph', text: 'Just text' }]);
    });

    test('a line splits into plain runs and code spans', () => {
        // Given - a line with two code spans, and two with a lone backtick
        const line = 'run `npm test` then `make`';
        const lone = ['`open', 'close`'];

        // Then - the runs, each marked; a lone backtick opens no span
        expect(inlineRuns(line)).toStrictEqual([
            { code: false, text: 'run ' },
            { code: true, text: 'npm test' },
            { code: false, text: ' then ' },
            { code: true, text: 'make' },
        ]);
        expect(lone.map((text) => inlineRuns(text))).toStrictEqual([
            [{ code: false, text: '`open' }],
            [{ code: false, text: 'close`' }],
        ]);
    });
});
