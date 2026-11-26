import { describe, expect, test } from 'vitest';

import { GEOMETRY } from '../geometry';
import type { CanvasNode } from '../graph';
import { measure } from '../place';
import type { Options } from '../place';

function node(depth: number, extra: Partial<CanvasNode> = {}): CanvasNode {
    return {
        band: 0,
        depth,
        id: 'n',
        kids: [],
        kind: 'any',
        label: 'label',
        marks: [],
        parent: null,
        ...extra,
    };
}

function options(labelWidth: number): Options {
    return { geometry: GEOMETRY, measureText: () => labelWidth };
}

describe('place', () => {
    test.each([
        ['a bare label, rounded up to the pixel', 57.2, 0, 103],
        ['a label and three marks, 9 px then 8 px each', 57.2, 3, 136],
        ['a label wider than 190 px, held to 190', 400, 0, 235],
        ['more marks than 11, held to 11', 57.2, 20, 200],
    ] as const)('a pill with %s is as wide as its parts', (_name, label, marks, width) => {
        // Given - a pill whose label measures `label` px and which carries `marks` marks
        const pill = node(2, { marks: Array.from({ length: marks }, () => false) });

        // When - it is measured
        const found = measure(pill, 'pill', options(label));

        // Then - its width is 33 px of chrome, the label, the marks and 12 px, and it is 30 px tall
        expect(found).toStrictEqual({ h: 30, w: width });
    });

    test.each([
        ['a depth-1 dot holds its three kids as a chip', 1, 3, { h: 24, w: 60 }],
        ['a depth-1 dot counts no more than ten kids', 1, 14, { h: 24, w: 116 }],
        ['a dot of another depth is 20 px', 2, 3, { h: 20, w: 20 }],
    ] as const)('%s', (_name, depth, kids, size) => {
        // Given - a node of that depth with that many kids
        const dot = node(depth, { kids: Array.from({ length: kids }, (_, index) => `k${index}`) });

        // Then - its dot is the size of its depth
        expect(measure(dot, 'dot', options(0))).toStrictEqual(size);
    });

    test.each([
        [2, { h: 118, w: 248 }],
        [3, { h: 116, w: 236 }],
        [4, { h: 108, w: 222 }],
        [5, { h: 108, w: 222 }],
    ] as const)('a card of depth %i has the size its depth is given', (depth, size) => {
        // Given - a node of that depth
        // Then - its card is the configured size, and a depth with none takes the last one
        expect(measure(node(depth), 'card', options(0))).toStrictEqual(size);
    });

    test('a label is measured by the function handed in, never by the module', () => {
        // Given - a measure that records what it was asked
        const asked: string[] = [];
        const spy: Options = {
            geometry: GEOMETRY,
            measureText: (text) => {
                asked.push(text);
                return 10;
            },
        };

        // When - a pill and a card are measured
        measure(node(2, { label: 'Boundaries' }), 'pill', spy);
        measure(node(2, { label: 'Boundaries' }), 'card', spy);

        // Then - only the pill asked, and it asked for its label
        expect(asked).toStrictEqual(['Boundaries']);
    });
});
