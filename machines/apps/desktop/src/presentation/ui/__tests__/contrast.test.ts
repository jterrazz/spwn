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

const GROUNDS = ['canvas', 'side', 'panel'] as const;

const TEXTS = ['t1', 't2', 't3'] as const;

const TONES = ['ok', 'warn', 'bad', 'new'] as const;

const TEXT_BAR = 4.5;

const MARK_BAR = 3;

function below(bar: number, measures: readonly Measure[]): string[] {
    return measures
        .filter(([, found]) => found < bar)
        .map(([name, found]) => `${name} ${found.toFixed(2)}`);
}

function overGrounds(of: string, over: (ground: string) => number): Measure[] {
    return GROUNDS.map((ground) => [`--${of} on --${ground}`, over(ground)]);
}

describe('contrast', () => {
    test('the three text levels read at 4.5:1 on each ground the window draws', () => {
        // Given - each text level of tokens.css on the canvas, the column and the card grounds
        const measures = TEXTS.flatMap((text) =>
            overGrounds(text, (ground) => ratio(colourOf(text), colourOf(ground))),
        );

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('a tone reads at 4.5:1 on its own pill over each ground', () => {
        // Given - each tone as text on its fill, the tone at its share over each ground
        const share = Number.parseFloat(tokenOf('tone-fill')) / 100;
        const measures = TONES.flatMap((tone) =>
            overGrounds(`${tone} on its pill over`, (ground) =>
                ratio(colourOf(tone), mixed(colourOf(ground), colourOf(tone), share)),
            ),
        );

        // Then - none is below the bar of text
        expect(below(TEXT_BAR, measures)).toStrictEqual([]);
    });

    test('the primary button label reads at 4.5:1 and the decorative level holds 3:1 on each ground', () => {
        // Given - the dark label on the light fill, and the decoration level on each ground
        const label: Measure = [
            '--on-primary on --primary',
            ratio(colourOf('on-primary'), colourOf('primary')),
        ];
        const decoration = overGrounds('t4', (ground) => ratio(colourOf('t4'), colourOf(ground)));

        // Then - the label holds the bar of text, and the decoration the bar of a mark
        expect(below(TEXT_BAR, [label])).toStrictEqual([]);
        expect(below(MARK_BAR, decoration)).toStrictEqual([]);
    });

    test('the pill is drawn at 8 % of its tone, and its line at 24 %', () => {
        // Given - the two shares tokens.css names
        const shares = [tokenOf('tone-fill'), tokenOf('tone-line')];

        // Then - they are the record's
        expect(shares).toStrictEqual(['8%', '24%']);
    });
});
