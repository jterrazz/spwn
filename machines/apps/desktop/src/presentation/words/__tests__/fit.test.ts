import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { FIT_REMEDY_WORDS, FIT_WORDS, fitFootWord, fitState } from '../fit';

const API_PAGE = join(
    import.meta.dirname,
    '..',
    '..',
    '..',
    '..',
    '..',
    '..',
    'docs',
    'reference',
    'api.md',
);

type Said = { verdict: string | undefined; remedy: string | undefined };

function quoted(cell: string | undefined): string | undefined {
    return /`(?<said>[^`]+)`/u.exec(cell ?? '')?.groups?.said;
}

function verdictTable(page: string): ReadonlyMap<string, Said> {
    const section = page.split('## The compatibility verdict')[1]?.split('\n## ')[0] ?? '';
    const rows = section
        .split('\n')
        .filter((line) => line.startsWith('| '))
        .map((line) => line.split('|').map((cell) => cell.trim()));
    return new Map(
        rows.map(([, revision = '', verdict, remedy]) => [
            revision.split(' ')[0] ?? '',
            { remedy: quoted(remedy), verdict: quoted(verdict) },
        ]),
    );
}

describe('fit', () => {
    test("the verdict's words are the reference page's, the shared vocabulary of both apps", () => {
        // Given - the table of docs/reference/api.md's compatibility verdict
        const table = verdictTable(readFileSync(API_PAGE, 'utf8'));

        // When - the window's words are read beside its rows
        const said = {
            above: FIT_WORDS['app-older'],
            below: { remedy: FIT_REMEDY_WORDS['engine-older'], verdict: FIT_WORDS['engine-older'] },
        };

        // Then - the older engine's verdict and remedy, and the older app's verdict, are word for word the page's
        expect(table.get('below')).toStrictEqual(said.below);
        expect(table.get('above')?.verdict).toBe(said.above);
    });

    test('the foot opens on the verdict itself, drawn as attention and never as stopped', () => {
        // Given - both verdicts

        // When - each is shown at the foot and as a state
        const shown = (['engine-older', 'app-older'] as const).map((fit) => [
            fitFootWord(fit),
            fitState(fit),
        ]);

        // Then - the same words, the line's first letter raised, in the warn tone
        expect(shown).toStrictEqual([
            ['Engine older than this app', { tone: 'warn', word: 'engine older than this app' }],
            [
                'This app older than the engine',
                { tone: 'warn', word: 'this app older than the engine' },
            ],
        ]);
    });
});
