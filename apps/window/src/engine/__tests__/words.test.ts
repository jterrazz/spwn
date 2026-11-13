import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, expect, test } from 'vitest';

const SRC = join(import.meta.dirname, '..', '..');

type Rule = { name: string; shape: RegExp };

// value: the patterns the law forbids outside words.ts, tones.ts, provisional.ts and shown.ts (README, The law):
// value: each is written to match no lawful line, so a finding is a mapping to move, never noise to silence.
const RULES: readonly Rule[] = [
    {
        name: 'a tone or a word written into a state component',
        shape: /<(?:StatePill|StateDot|Verdict)\b[^<>]*?\b(?:tone|word)=(?:"|\{\s*['"`])/gu,
    },
    { name: 'a tone written into a state', shape: /\btone:\s*['"`]/gu },
    {
        name: 'a wire field compared with one of its codes',
        shape: /\.[A-Z]\w*\s*[!=]==\s*'[^']+'|'[^']+'\s*[!=]==\s*[\w?.]*\.[A-Z]\w*\b/gu,
    },
    {
        name: 'a table keyed by a wire code',
        shape: /Record<\s*(?:NonNullable<\s*)?[A-Z]\w*\['\w+'\]/gu,
    },
];

type Exempt = { file: RegExp; rule: string; why: string };

const EXEMPT: readonly Exempt[] = [
    {
        file: /^pages\/gallery\//u,
        rule: '',
        why: 'the gallery draws every design component with samples, never an engine answer',
    },
    {
        file: /^pages\/needs\/needs\.ts$/u,
        rule: 'a table keyed by a wire code',
        why: "a need's section to the machine tab it opens is routing (README, the engine-owes table)",
    },
];

function sourcesUnder(folder: string): string[] {
    return readdirSync(join(SRC, folder), { recursive: true, withFileTypes: true })
        .filter((entry) => entry.isFile() && /\.tsx?$/u.test(entry.name))
        .map((entry) => relative(SRC, join(entry.parentPath, entry.name)))
        .filter((path) => !path.includes('__tests__'));
}

function isExempt(file: string, rule: string): boolean {
    return EXEMPT.some((one) => one.file.test(file) && (one.rule === '' || one.rule === rule));
}

function lineAt(text: string, index: number): number {
    return text.slice(0, index).split('\n').length;
}

function findingsIn(file: string, text: string): string[] {
    return RULES.filter((rule) => !isExempt(file, rule.name)).flatMap((rule) =>
        [...text.matchAll(rule.shape)].map(
            (found) => `${file}:${String(lineAt(text, found.index))} ${rule.name}`,
        ),
    );
}

function foundIn(lines: readonly string[]): number[] {
    return lines.map((line) => findingsIn('pages/any.tsx', line).length);
}

describe('words', () => {
    test('no page and no part of the shell maps a code of its own', () => {
        // Given - every source of src/pages/ and src/shell/ but the specs
        const files = [...sourcesUnder('pages'), ...sourcesUnder('shell')];

        // When - each is read for the patterns the law forbids
        const findings = files.flatMap((file) =>
            findingsIn(file, readFileSync(join(SRC, file), 'utf8')),
        );

        // Then - none is found
        expect(files.length).toBeGreaterThan(50);
        expect(findings).toStrictEqual([]);
    });

    test('each pattern finds the mapping it names, and none a lawful line', () => {
        // Given - one forbidden line per rule, and the lawful lines nearest to each
        const forbidden = [
            '<StatePill tone="warn" word={WORDS[port.Answer]} />',
            '<Verdict line={line} size="tab" title="t" word="Not given" />',
            "state={{ tone: 'ok', word: 'settled' }}",
            "return need.Section === 'ports';",
            "const GESTURES: Readonly<Record<DiskRow['Group'], Gesture>> = {};",
        ];
        const lawful = [
            '<StatePill tone={TONES[port.Answer]} word={WORDS[port.Answer]} />',
            '<Verdict line={line} size="tab" title="t" {...ownState(\'not-given\')} />',
            '<Chart label={label} marks={marks} size="activity" tone="new" values={values} />',
            "if (port.Answer === '') {",
            "if (reading.kind === 'failed') {",
            "return route.page === 'machine';",
            'const TITLES: Record<PlainPage, string> = {};',
        ];

        // When - each is read as a page's line
        const found = [foundIn(forbidden), foundIn(lawful)];

        // Then - every forbidden line is found once, no lawful one
        expect(found).toStrictEqual([
            [1, 1, 1, 1, 1],
            [0, 0, 0, 0, 0, 0, 0],
        ]);
    });
});
