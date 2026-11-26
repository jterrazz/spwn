import { isDeepStrictEqual } from 'node:util';
import { describe, expect, test } from 'vitest';

import { GEOMETRY } from '../geometry';
import type { Placed, Rect } from '../graph';
import { layout } from '../layout';
import type { Layout } from '../layout';
import type { Options } from '../place';
import {
    BOUNDARIES,
    facesOf,
    GRAPH,
    OPTIONS,
    PRODUIT,
    stateOf,
    STATES,
    WHAT_IT_IS_NOT,
    WIDE,
} from './prototype';
import type { ProtoState } from './prototype';

function laidOut(state: ProtoState, options: Options = OPTIONS): Layout {
    return layout(GRAPH, state.sel, state.viewport, options);
}

function rectOf(found: Layout, id: string): Placed {
    const rect = found.rects.get(id);
    if (rect === undefined) {
        throw new Error(`no rect for ${id}`);
    }
    return rect;
}

function seen(found: Layout): Record<string, Placed> {
    return Object.fromEntries(found.rects);
}

function overlaps(one: Rect, other: Rect): boolean {
    return (
        one.x < other.x + other.w &&
        other.x < one.x + one.w &&
        one.y < other.y + other.h &&
        other.y < one.y + one.h
    );
}

function inside(inner: Rect, outer: Rect): boolean {
    return (
        inner.x >= outer.x &&
        inner.y >= outer.y &&
        inner.x + inner.w <= outer.x + outer.w &&
        inner.y + inner.h <= outer.y + outer.h
    );
}

function the(
    id: string,
    face: Placed['face'],
    [x, y, w, h]: readonly [number, number, number, number],
) {
    return [id, { face, h, w, x, y }] as const;
}

function floorGap(found: Layout): number {
    return (found.guides[1]?.y ?? 0) - (found.guides[0]?.y ?? 0);
}

const AT_ROOT = [
    the('S:blueprint', 'region', [132, 0, 280, 180]),
    the('P:blueprint/philosophy', 'card', [148, 46, 248, 118]),
    the('S:product', 'region', [132, 232, 800, 180]),
    the('P:product/boundaries', 'card', [148, 278, 248, 118]),
    the('P:product/stack', 'card', [408, 278, 248, 118]),
    the('S:growth', 'region', [960, 232, 280, 180]),
    the('P:growth/positioning', 'card', [976, 278, 248, 118]),
    the('S:engineering', 'region', [132, 464, 540, 180]),
    the('P:machines/reference/ports', 'card', [148, 510, 248, 118]),
] as const;

const A_SEAT_OPEN = [
    the('S:blueprint', 'pill', [132, 0, 100, 30]),
    the('S:product', 'open', [132, 70, 800, 180]),
    the('P:product/boundaries', 'card', [148, 116, 248, 118]),
    the('P:product/stack', 'card', [408, 116, 248, 118]),
    the('S:growth', 'pill', [960, 70, 129, 30]),
    the('S:engineering', 'pill', [132, 290, 132, 30]),
] as const;

const A_PAGE_OPEN = [
    the('S:blueprint', 'dot', [132, 0, 44, 24]),
    the('S:product', 'open', [132, 70, 1100, 318]),
    the('P:product/stack', 'pill', [148, 116, 137, 30]),
    the('P:product/boundaries', 'open', [148, 160, 1012, 212]),
    the('C:product/boundaries:0', 'card', [164, 240, 236, 116]),
    the('C:product/boundaries:3', 'card', [908, 240, 236, 116]),
    the('S:growth', 'dot', [1260, 70, 44, 24]),
    the('S:engineering', 'dot', [132, 428, 52, 24]),
] as const;

const A_SECTION_OPEN = [
    the('P:product/stack', 'dot', [148, 121, 20, 20]),
    the('S:product', 'open', [132, 70, 1100, 536]),
    the('P:product/boundaries', 'open', [148, 160, 1068, 430]),
    the('C:product/boundaries:0', 'pill', [164, 240, 84, 30]),
    the('C:product/boundaries:2', 'pill', [256, 240, 256, 30]),
    the('C:product/boundaries:1', 'open', [164, 284, 956, 290]),
    the('S:engineering', 'dot', [132, 646, 52, 24]),
] as const;

describe('layout', () => {
    test.each([
        ['the root, every seat a region holding its pages (c1)', null, AT_ROOT],
        ['one seat open, the others pills (c3)', PRODUIT, A_SEAT_OPEN],
        [
            'one page open, its siblings pills and the other seats dots (c4)',
            BOUNDARIES,
            A_PAGE_OPEN,
        ],
        [
            'one section open, its siblings pills, the pages dots (c5)',
            WHAT_IT_IS_NOT,
            A_SECTION_OPEN,
        ],
    ] as const)('%s lays the prototype rects out to the pixel', (_name, id, rows) => {
        // Given - the prototype's graph and its label widths, at the 1440 window the captures show
        const state = stateOf(WIDE, id);

        // When - the open path is laid out
        const found = laidOut(state);

        // Then - the rows the capture shows are exactly where the prototype put them
        expect(rows.map(([rowId]) => rectOf(found, rowId))).toStrictEqual(
            rows.map(([, rect]) => rect),
        );
    });

    test('every state the prototype was driven through reproduces its rects, guides and box with no pixel of difference', () => {
        // Given - 47 states: the root, each seat, page and section at 1440, and the root, seats and pages at 1100
        const differences = STATES.flatMap((state) => {
            // When - the path of the state is laid out in its viewport
            const found = laidOut(state);

            // Then - the three outputs equal the prototype's dump
            const same =
                isDeepStrictEqual(seen(found), state.rects) &&
                isDeepStrictEqual(found.guides, state.guides) &&
                isDeepStrictEqual(found.box, state.box);
            return same ? [] : [`${state.id ?? 'root'} at ${state.viewport.width}`];
        });
        expect(STATES).toHaveLength(47);
        expect(differences).toStrictEqual([]);
    });

    test('the prototype dump and the layout hold the same nodes, with the same faces, and the same floating-point numbers', () => {
        // Given - the section state, the deepest the captures show
        const state = stateOf(WIDE, WHAT_IT_IS_NOT);

        // When - it is laid out
        const found = laidOut(state);

        // Then - the rects, key by key, are the dump's, and so are the faces
        expect(seen(found)).toStrictEqual(state.rects);
        expect(facesOf(found.rects)).toStrictEqual(facesOf(Object.entries(state.rects)));
        expect(found.guides).toStrictEqual(state.guides);
        expect(found.box).toStrictEqual(state.box);
    });

    test('no two rects of one row of siblings overlap, in any state', () => {
        // Given - every state's rects, grouped by the parent that holds them
        const clashes = STATES.flatMap((state) => {
            const found = laidOut(state);
            const byParent = Map.groupBy(
                [...found.rects.keys()],
                (id) => GRAPH.nodes.get(id)?.parent ?? 'root',
            );
            return [...byParent.values()].flatMap((siblings) =>
                siblings.flatMap((one, index) =>
                    siblings
                        .slice(index + 1)
                        .filter((other) => overlaps(rectOf(found, one), rectOf(found, other)))
                        .map((other) => `${one} x ${other} in ${state.id ?? 'root'}`),
                ),
            );
        });

        // Then - none touches another
        expect(clashes).toStrictEqual([]);
    });

    test('every child rect lies inside the rect of the open node that holds it, in any state', () => {
        // Given - every placed node whose parent is placed open or as a region
        const strays = STATES.flatMap((state) => {
            const found = laidOut(state);
            return [...found.rects].flatMap(([id]) => {
                const parent = GRAPH.nodes.get(id)?.parent;
                const holder =
                    parent === null || parent === undefined ? undefined : found.rects.get(parent);
                const held = holder?.face === 'open' || holder?.face === 'region';
                return held && holder !== undefined && !inside(rectOf(found, id), holder)
                    ? [`${id} in ${state.id ?? 'root'}`]
                    : [];
            });
        });

        // Then - none leaves its holder
        expect(strays).toStrictEqual([]);
    });

    test('the seats of one floor share a row, the floors descend, and a guide labels each', () => {
        // Given - the root, where Produit and Croissance are floor 1 and Technique floor 3
        const found = laidOut(stateOf(WIDE, null));

        // Then - the floor 1 seats share a top, the floors are listed in order, and a guide stands above each row
        expect(rectOf(found, 'S:product').y).toBe(rectOf(found, 'S:growth').y);
        expect(found.guides.map((guide) => guide.band)).toStrictEqual([0, 1, 3]);
        expect(found.guides.map((guide) => guide.y)).toStrictEqual([-20, 212, 444]);
    });

    test('a node open tightens the floors by 12 px, from 52 to 40', () => {
        // Given - the root and a seat open
        const closed = laidOut(stateOf(WIDE, null));
        const open = laidOut(stateOf(WIDE, 'S:blueprint'));

        // Then - the second floor's guide sits 12 px nearer the first
        expect(floorGap(closed) - floorGap(open)).toBe(
            GEOMETRY.layout.bandGap - GEOMETRY.layout.bandGapOpen,
        );
    });

    test('the layout is idempotent: the same graph, path and viewport give the same rects', () => {
        // Given - one state laid out twice
        const state = stateOf(WIDE, BOUNDARIES);

        // When - it is laid out again, and after another path in between
        const first = laidOut(state);
        laidOut(stateOf(WIDE, null));
        const second = laidOut(state);

        // Then - nothing differs
        expect(second).toStrictEqual(first);
    });
});
