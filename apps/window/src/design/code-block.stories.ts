import type { CodeBlockProps } from './code-block';
import type { Story } from './story';

const MANY = Array.from({ length: 60 }, (_, index) => ({
    number: index + 1,
    text: `2026-11-13 09:${String(index).padStart(2, '0')}:00 relay check ok`,
}));

export const CODE_BLOCK_STORIES: readonly Story<CodeBlockProps>[] = [
    {
        name: 'plain',
        props: {
            label: 'A file',
            lines: [
                { number: 1, text: 'name: jterrazz-base' },
                { number: 2, text: 'apps: [jq, git, node]' },
            ],
        },
    },
    {
        name: 'a diff, numbered',
        props: {
            label: 'A diff',
            lines: [
                { number: 1, text: 'extends: [base, dev-tools]' },
                { number: 2, text: 'apps:' },
                { change: 'added', number: 3, text: '  - lazygit' },
                { change: 'removed', number: 4, text: '  - jq' },
            ],
            numbered: true,
        },
    },
    {
        name: 'a log, windowed, with its foot',
        props: {
            foot: 'The last 60 lines',
            label: 'A log',
            lines: MANY,
            log: true,
            numbered: true,
        },
    },
    {
        name: 'long text',
        props: {
            label: 'A long line',
            lines: [{ number: 1, text: `export PATH=${'/opt/homebrew/bin:'.repeat(12)}$PATH` }],
        },
    },
    {
        name: 'empty',
        props: { empty: 'No line yet.', label: 'An empty log', lines: [], log: true },
    },
];
