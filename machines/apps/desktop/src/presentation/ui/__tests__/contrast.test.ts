import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

import { mixed, ratio, rgbOf } from '../contrast';

const TOKENS = readFileSync(join(import.meta.dirname, '..', 'tokens.css'), 'utf8');

const DARK = TOKENS.slice(0, TOKENS.indexOf("html[data-material='vibrant']"));

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

const TONES = ['ok', 'you', 'bad', 'busy'] as const;

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

    test("a state's mark holds 3:1 on the column's current row, the light pill", () => {
        // Given - each tone's mark as the current row draws it, on --primary
        const measures = ['ok', 'warn', 'bad', 'new'].map((tone): Measure => [
            `--${tone}-on-primary on --primary`,
            ratio(colourOf(`${tone}-on-primary`), colourOf('primary')),
        ]);

        // Then - none is below the bar of a mark
        expect(below(MARK_BAR, measures)).toStrictEqual([]);
    });

    test('a tone reads at 4.5:1 on its own pill over s0 to s2, the steps a pill sits on', () => {
        // Given - each tone as text on its fill, the tone at its share over s0 to s2
        const share = Number.parseFloat(tokenOf('tone-fill')) / 100;
        const measures = TONES.flatMap((tone) =>
            overGrounds(
                `${tone} on its pill over`,
                (ground) => ratio(colourOf(tone), mixed(colourOf(ground), colourOf(tone), share)),
                QUIET_GROUNDS,
            ),
        );

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('the pill is drawn at 8 % of its tone, and its line at 24 %', () => {
        // Given - the two shares tokens.css names
        const shares = [tokenOf('tone-fill'), tokenOf('tone-line')];

        // Then - they are the record's
        expect(shares).toStrictEqual(['8%', '24%']);
    });
});
