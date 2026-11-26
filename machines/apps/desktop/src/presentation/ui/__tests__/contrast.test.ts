import { readdirSync, readFileSync } from 'node:fs';
import { join, relative } from 'node:path';
import { describe, expect, test } from 'vitest';

import { mixed, ratio, rgbOf } from '../contrast';
import type { Rgb } from '../contrast';

const TOKENS = readFileSync(join(import.meta.dirname, '..', 'tokens.css'), 'utf8');

const PRESENTATION = join(import.meta.dirname, '..', '..');

const SCALES = ['text', 'radius', 'space', 'motion'] as const;

type Sheet = { path: string; source: string };

function sheetsUnder(dir: string): Sheet[] {
    return readdirSync(dir, { withFileTypes: true }).flatMap((entry): Sheet[] => {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) {
            return sheetsUnder(path);
        }
        return path.endsWith('.css') ? [{ path, source: readFileSync(path, 'utf8') }] : [];
    });
}

const SHEETS = sheetsUnder(PRESENTATION);

const STYLES = SHEETS.map(({ source }) => source).join('\n');

function sheetsDrawing(token: string): string[] {
    const drawn = new RegExp(String.raw`var\(--${token}[,)]`, 'u');
    return SHEETS.filter(({ source }) => drawn.test(source))
        .map(({ path }) => path.slice(PRESENTATION.length + 1))
        .toSorted();
}

type Rule = { file: string; selector: string; body: string };

const RULES: readonly Rule[] = SHEETS.flatMap(({ path, source }) =>
    [...source.matchAll(/(?<selector>[^{}]+)\{(?<body>[^{}]*)\}/gu)].map((found) => ({
        body: found.groups?.body ?? '',
        file: relative(PRESENTATION, path),
        selector: (found.groups?.selector ?? '').trim(),
    })),
);

const GLASS = RULES.filter((rule) => rule.body.includes('var(--glass)'));

function declares(body: string, property: string, value: string): boolean {
    const escaped = value.replaceAll(/[()]/gu, String.raw`\$&`);
    return new RegExp(String.raw`(?:^|[;\s])${property}:\s*${escaped}`, 'u').test(body);
}

function scaleNames(source: string, around: 'declared' | 'drawn'): Set<string> {
    const name = `--(?:${SCALES.join('|')})-[a-z0-9-]+`;
    const pattern = around === 'drawn' ? `var\\((${name})` : `(${name}):`;
    return new Set([...source.matchAll(new RegExp(pattern, 'gu'))].map((each) => each[1] ?? ''));
}

const DARK = TOKENS.slice(0, TOKENS.indexOf('@media (prefers-reduced-transparency: reduce)'));

function tokenOf(name: string): string {
    const found = new RegExp(`--${name}:\\s*(?<value>[^;]+);`, 'u').exec(DARK);
    if (found?.groups === undefined) {
        throw new Error(`tokens.css holds no --${name}`);
    }
    return (found.groups.value ?? '').trim();
}

function colourOf(name: string) {
    return rgbOf(tokenOf(name));
}

type Measure = readonly [name: string, ratio: number];

const GROUNDS = ['s0', 's1', 's2', 's3'] as const;

const QUIET_GROUNDS = ['s0', 's1', 's2'] as const;

const TEXTS = ['t1', 't2', 't3'] as const;

const MARKS = ['ok', 'bad', 'you', 'busy', 't3'] as const;

const TEXT_BAR = 4.5;

const MARK_BAR = 3;

function below(bar: number, measures: readonly Measure[]): string[] {
    return measures
        .filter(([, found]) => found < bar)
        .map(([name, found]) => `${name} ${found.toFixed(2)}`);
}

function overGrounds(
    of: string,
    over: (ground: string) => number,
    grounds: readonly string[] = GROUNDS,
): Measure[] {
    return grounds.map((ground) => [`--${of} on --${ground}`, over(ground)]);
}

function onGrounds(name: string, grounds: readonly string[] = GROUNDS): Measure[] {
    return overGrounds(name, (ground) => ratio(colourOf(name), colourOf(ground)), grounds);
}

function glassOver(ground: string): Rgb {
    const found = /^rgb\((?<red>\d+) (?<green>\d+) (?<blue>\d+) \/ (?<share>\d+)%\)$/u.exec(
        tokenOf('glass'),
    );
    if (found?.groups === undefined) {
        throw new Error(`--glass is not an rgb() with a share: ${tokenOf('glass')}`);
    }
    const { blue, green, red, share } = found.groups;
    const tint = { blue: Number(blue), green: Number(green), red: Number(red) };
    return mixed(colourOf(ground), tint, Number(share) / 100);
}

function numberOf(name: string): number {
    return Number(tokenOf(name));
}

function translucent(name: string, over: Rgb): Rgb {
    const found = /^rgb\((?<red>\d+) (?<green>\d+) (?<blue>\d+) \/ (?<share>[\d.]+)%\)$/u.exec(
        tokenOf(name),
    );
    if (found?.groups === undefined) {
        throw new Error(`--${name} is not an rgb() with a share: ${tokenOf(name)}`);
    }
    const { blue, green, red, share } = found.groups;
    const tint = { blue: Number(blue), green: Number(green), red: Number(red) };
    return mixed(over, tint, Number(share) / 100);
}

function onGlass(text: string, grounds: readonly string[]): Measure[] {
    return grounds.map((ground) => [
        `--${text} on --glass over --${ground}`,
        ratio(colourOf(text), glassOver(ground)),
    ]);
}

describe('contrast', () => {
    test('the three text levels read at 4.5:1 on each of the four surface steps', () => {
        // Given - each text level of tokens.css on s0 to s3
        const measures = TEXTS.flatMap((text) => onGrounds(text));

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('the decorative level holds 3:1 on s0 to s2, and s3 is the step it is never drawn on', () => {
        // Given - the decoration level on the three steps it may sit on, and on s3
        const decoration = onGrounds('t4', QUIET_GROUNDS);
        const onS3: Measure = ['--t4 on --s3', ratio(colourOf('t4'), colourOf('s3'))];

        // Then - it holds the bar of a mark on the three, and falls under it on s3
        expect(below(MARK_BAR, decoration)).toStrictEqual([]);
        expect(below(MARK_BAR, [onS3])).toStrictEqual([`--t4 on --s3 ${onS3[1].toFixed(2)}`]);
    });

    test('each mark holds the bar of a mark on each of the four surface steps', () => {
        // Given - the five marks of a state on s0 to s3
        const measures = MARKS.flatMap((mark) => onGrounds(mark));

        // Then - none is below the bar of a mark
        expect(below(MARK_BAR, measures)).toStrictEqual([]);
    });

    test('the two colours drawn as text read at 4.5:1 on each of the four surface steps', () => {
        // Given - the orange and the red as text on s0 to s3
        const measures = ['you', 'bad'].flatMap((text) => onGrounds(text));

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('a label on its fill reads at 4.5:1: the primary, the destroy, the orange, and a control at hover', () => {
        // Given - each dark label on its fill, and the text on the hovered control
        const labels: Measure[] = [
            ['--on-primary on --primary', ratio(colourOf('on-primary'), colourOf('primary'))],
            ['--on-bad on --bad', ratio(colourOf('on-bad'), colourOf('bad'))],
            ['--on-you on --you', ratio(colourOf('on-you'), colourOf('you'))],
            ['--t1 on --s3-hover', ratio(colourOf('t1'), colourOf('s3-hover'))],
        ];

        // Then - each holds the bar of text
        expect(below(TEXT_BAR, labels)).toStrictEqual([]);
    });

    test('the destroy button draws its label in --on-bad, on its fill and while disabled', () => {
        // Given - the destroy rules of the button's style sheet
        const button = readFileSync(join(import.meta.dirname, '..', 'button.module.css'), 'utf8');
        const colours = [
            ...button.matchAll(/\.destroy(?::disabled)? \{[^}]*?\bcolor: (?<colour>[^;]+);/gu),
        ].map((each) => each.groups?.colour);

        // Then - both rules set the dark label the contrast table measures
        expect(colours).toStrictEqual(['var(--on-bad)', 'var(--on-bad)']);
    });

    test('orange is drawn by the style sheets that carry a focus, a need, a warning or the disk running out, and no other', () => {
        // Given - every style sheet that draws --you
        const orange = sheetsDrawing('you');

        // Then - the closed list: a new sheet that draws the owner's colour is a deliberate edit of this one
        expect(orange).toStrictEqual([
            'features/disk/disk.module.css',
            'kit/login-sheet.module.css',
            'navigation/column.module.css',
            'navigation/path-bar.module.css',
            'screens/catalog/item.module.css',
            'screens/machine-files/machine-files.module.css',
            'screens/machine/overview/overview.module.css',
            'screens/machine/packages/packages.module.css',
            'screens/machine/storage/machine-storage.module.css',
            'screens/passwords/passwords.module.css',
            'ui/bar.module.css',
            'ui/chart.module.css',
            'ui/document.css',
            'ui/dot-strip.module.css',
            'ui/fleet-board.module.css',
            'ui/form-field.module.css',
            'ui/guide.module.css',
            'ui/illustrations/illustration.module.css',
            'ui/nav-item.module.css',
            'ui/search-field.module.css',
            'ui/state-mark.module.css',
            'ui/switch.module.css',
            'ui/tile.module.css',
            'ui/value.module.css',
        ]);
    });

    test('blue is drawn by the style sheets of a busy mark, a progress line or a running step, and no other', () => {
        // Given - every style sheet that draws --busy
        const blue = sheetsDrawing('busy');

        // Then - the closed list: no focus, link, selection or identity is blue
        expect(blue).toStrictEqual([
            'screens/core/core.module.css',
            'ui/bar.module.css',
            'ui/chart.module.css',
            'ui/dot-strip.module.css',
            'ui/progress-line.module.css',
            'ui/state-mark.module.css',
            'ui/tile.module.css',
        ]);
    });

    test('the retired amber, blue and orange tokens are drawn by no style sheet', () => {
        // Given - the old warn, new and accent tokens
        const retired = ['warn', 'new', 'accent'].flatMap((token) => sheetsDrawing(token));

        // Then - none remains
        expect(retired).toStrictEqual([]);
    });

    test('text on glass reads at 4.5:1 over each of the four surface steps', () => {
        // Given - each text level on the glass composited over s0 to s3, the blur left out since it only averages
        const measures = TEXTS.flatMap((text) => onGlass(text, GROUNDS));

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('over the white primary button, the brightest thing glass floats over, a name and its meta keep 4.5:1 and the third level 3:1', () => {
        // Given - the text levels on the glass composited over the primary at rest and at hover, pure white
        const grounds = ['primary', 'primary-hover'];
        const read = [...onGlass('t1', grounds), ...onGlass('t2', grounds)];
        const third = onGlass('t3', grounds);

        // Then - none falls under its bar
        expect(below(TEXT_BAR, read)).toStrictEqual([]);
        expect(below(MARK_BAR, third)).toStrictEqual([]);
    });

    test('a control one step above its surface differs from it by 1.06:1 at least, the path bar to its Find box included', () => {
        // Given - each surface step against the step a control on it takes
        const steps: Measure[] = [
            ['--s1 on --s0', ratio(colourOf('s1'), colourOf('s0'))],
            ['--s2 on --s1 (the bar and its Find)', ratio(colourOf('s2'), colourOf('s1'))],
            ['--s3 on --s2', ratio(colourOf('s3'), colourOf('s2'))],
        ];

        // Then - none is closer than the plan's lowest step
        expect(below(1.06, steps)).toStrictEqual([]);
    });

    test('the drawings ink holds the bar of a mark on each of the four surface steps', () => {
        // Given - the drawings' line on s0 to s3
        const marks = [...onGrounds('art-ink'), ...onGrounds('art-dim')];

        // Then - each holds the bar of a mark
        expect(below(MARK_BAR, marks)).toStrictEqual([]);
    });

    test('on the canvas ground and on a region, the three text levels read at 4.5:1 and the decorative level at 3:1', () => {
        // Given - each text level on the desk and on the region composited over it, and the level that labels a floor's guide
        const region = translucent('region', colourOf('desk'));
        const read: Measure[] = TEXTS.flatMap((text) => [
            ...onGrounds(text, ['desk']),
            [`--${text} on --region over --desk`, ratio(colourOf(text), region)],
        ]);

        // Then - each holds its bar
        expect(below(TEXT_BAR, read)).toStrictEqual([]);
        expect(below(MARK_BAR, onGrounds('t4', ['desk']))).toStrictEqual([]);
    });

    test('a pill left behind on the canvas keeps its word at 3:1 while it recedes over the desk', () => {
        // Given - the word on each pill surface, both faded by --dim-recede over the desk
        const share = numberOf('dim-recede');
        const faded = (colour: Rgb) => mixed(colourOf('desk'), colour, share);
        const measures: Measure[] = ['s2', 's3'].flatMap((ground) =>
            ['t1', 't2'].map((text): Measure => [
                `--${text} on --${ground} at ${share}`,
                ratio(faded(colourOf(text)), faded(colourOf(ground))),
            ]),
        );

        // Then - none is below the bar of a mark
        expect(below(MARK_BAR, measures)).toStrictEqual([]);
    });

    test('the unread part of a page keeps its text at 3:1 while --dim-unread fades it over the page', () => {
        // Given - the two readable levels faded by --dim-unread over s0
        const share = numberOf('dim-unread');
        const measures: Measure[] = ['t1', 't2'].map((text) => [
            `--${text} on --s0 at ${share}`,
            ratio(mixed(colourOf('s0'), colourOf(text), share), colourOf('s0')),
        ]);

        // Then - none is below the bar of a mark
        expect(below(MARK_BAR, measures)).toStrictEqual([]);
    });

    test('every scale token a rule draws is declared in tokens.css, and every one declared is drawn', () => {
        // Given - the text, radius, space and motion tokens the style sheets draw, and the ones tokens.css declares
        const drawn = scaleNames(STYLES, 'drawn');
        const declared = scaleNames(TOKENS, 'declared');

        // Then - none is drawn undeclared, and none stands unused, so a renamed step leaves nothing behind
        expect([...drawn].filter((name) => !declared.has(name))).toStrictEqual([]);
        expect([...declared].filter((name) => !drawn.has(name))).toStrictEqual([]);
    });

    test('what floats wears glass, and nothing else does: no page, tile or sheet', () => {
        // Given - every rule of the style sheets that draws the glass
        const worn = GLASS.map((rule) => `${rule.file} ${rule.selector}`).toSorted();

        // Then - they are the floating surfaces, and only they
        expect(worn).toStrictEqual([
            'navigation/find.module.css .pop',
            'ui/popover.module.css .popover',
        ]);
    });

    test('a glass rule draws the blur, its WebKit twin, an edge and the floating lift', () => {
        // Given - every rule that draws the glass, and the parts of the material each must draw
        const parts: readonly (readonly [string, (body: string) => boolean])[] = [
            ['backdrop-filter', (body) => declares(body, 'backdrop-filter', 'var(--blur)')],
            [
                '-webkit-backdrop-filter',
                (body) => declares(body, '-webkit-backdrop-filter', 'var(--blur)'),
            ],
            ['edge', (body) => body.includes('var(--edge)')],
            ['lift', (body) => declares(body, 'box-shadow', 'var(--lift-2)')],
        ];
        const missing = GLASS.flatMap((rule) =>
            parts
                .filter(([, drawn]) => !drawn(rule.body))
                .map(([part]) => `${rule.file} ${rule.selector}: ${part}`),
        );

        // Then - none lacks a part of the material
        expect(missing).toStrictEqual([]);
    });

    test('a backdrop filter is drawn by glass alone', () => {
        // Given - every rule that blurs what lies under it
        const blurring = RULES.filter((rule) => rule.body.includes('backdrop-filter:'));

        // Then - each is a glass rule
        expect(blurring.filter((rule) => !GLASS.includes(rule))).toStrictEqual([]);
    });

    test('asked for less transparency, glass becomes the opaque s2 step and blurs nothing', () => {
        // Given - the tokens tokens.css sets when the system asks for reduced transparency
        const reduced = TOKENS.slice(
            TOKENS.indexOf('@media (prefers-reduced-transparency: reduce)'),
        ).split('}')[0];

        // Then - the glass is s2 and the blur is none, and nothing else changes there
        expect(reduced?.match(/--[a-z0-9-]+:[^;]+;/gu)).toStrictEqual([
            '--glass: var(--s2);',
            '--blur: none;',
        ]);
    });
});
