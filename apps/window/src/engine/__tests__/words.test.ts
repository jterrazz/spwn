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

const API = join(SRC, '..', '..', '..', 'packages', 'api');

const DECLARED = join(API, 'index.d.ts');

const ENGINE_READER = 'engine/engine-words.ts';

type Enums = ReadonlyMap<string, string>;

function codeSet(codes: readonly string[]): string {
    return codes.toSorted((one, other) => one.localeCompare(other)).join('|');
}

function keysOf(value: unknown): string[] {
    return typeof value === 'object' && value !== null ? Object.keys(value) : [];
}

function wordedEnums(): Enums {
    const words: unknown = JSON.parse(readFileSync(join(API, 'words.json'), 'utf8'));
    const tables = typeof words === 'object' && words !== null ? Object.entries(words) : [];
    return new Map(
        tables.map(([name, table]: [string, unknown]) => [name, codeSet(keysOf(table))]),
    );
}

function fieldsOf(enums: Enums, declared: string): string[] {
    const covered = new Set(enums.values());
    const fields: string[] = [];
    let owner = '';
    for (const line of declared.split('\n')) {
        owner = /^export type (?<owner>\w+) = \{$/u.exec(line)?.groups?.owner ?? owner;
        const field = /^\s+(?<name>\w+)\??: (?<type>[^;]+);$/u.exec(line)?.groups;
        const type = field?.type ?? '';
        const codes = [...type.matchAll(/'(?<code>[^']*)'/gu)]
            .map((found) => found.groups?.code ?? '')
            .filter((code) => code !== '');
        const rest = type.replaceAll(/'[^']*'/gu, '').replaceAll(/null|[\s()|[\]]/gu, '');
        if (field !== undefined && rest === '' && covered.has(codeSet(codes))) {
            fields.push(`${owner}\\['${field.name ?? ''}'\\]`);
        }
    }
    return fields;
}

function tableOfAWordedEnum(enums: Enums, declared: string): RegExp {
    const keys = [...enums.keys(), ...fieldsOf(enums, declared)].join('|');
    return new RegExp(
        String.raw`(?:Words|Tones|Record)<\s*(?:''\s*\|\s*)?(?:NonNullable<\s*)?(?:${keys})(?!\w)`,
        'gu',
    );
}

const WORDED_TABLE = tableOfAWordedEnum(wordedEnums(), readFileSync(DECLARED, 'utf8'));

function wordedTablesIn(file: string, text: string): string[] {
    return [...text.matchAll(WORDED_TABLE)].map(
        (found) => `${file}:${String(lineAt(text, found.index))}`,
    );
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

    test("no table words or tones an enum the engine words, the engine's reader aside", () => {
        // Given - every source of src/ but the fixtures and the specs, the engine's words read by one module
        const files = ['engine', 'pages', 'shell', 'model', 'design']
            .flatMap((folder) => sourcesUnder(folder))
            .filter((file) => file !== ENGINE_READER);

        // When - each is read for a table keyed by an enum of packages/api/words.json
        const findings = files.flatMap((file) =>
            wordedTablesIn(file, readFileSync(join(SRC, file), 'utf8')),
        );

        // Then - none is found
        expect(files.length).toBeGreaterThan(100);
        expect(findings).toStrictEqual([]);
    });

    test('a table of an enum the engine words is found by its name or by its field, and no other', () => {
        // Given - tables of the enums words.json covers, and the lawful lines nearest to them
        const forbidden = [
            'const L: Words<MachinesLifetime> = {};',
            "const L: Words<MachinesVM['Lifetime']> = {};",
            "const T: Tones<MachinefileOrigin['Layer']> = {};",
            "const S: Words<MachinesVM['Kind']> = {};",
            "const C: Words<MachinesBirth['step']> = {};",
            "const P: Tones<MachinefileChapterDrift['Chapter']> = {};",
            "const K: Readonly<Record<NonNullable<CatalogMethodSpec['OS']>, string>> = {};",
        ];
        const lawful = [
            "const S: Words<MachinesVM['State']> = {};",
            "const K: Words<NonNullable<CatalogApp['Kind']>> = {};",
            "export const LIFETIMES = ['ephemeral', 'persistent'] as const satisfies readonly MachinesLifetime[];",
        ];

        // When - each is read as a line of a page
        const found = [forbidden, lawful].map((lines) =>
            lines.map((line) => wordedTablesIn('pages/any.tsx', line).length),
        );

        // Then - every forbidden line is found once, no lawful one
        expect(found).toStrictEqual([
            [1, 1, 1, 1, 1, 1, 1],
            [0, 0, 0],
        ]);
    });
});
