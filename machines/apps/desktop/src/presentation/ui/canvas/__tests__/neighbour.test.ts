import { describe, expect, test } from 'vitest';

import { cameraShowing } from '../camera';
import { GEOMETRY } from '../geometry';
import type { Rect } from '../graph';
import { layout } from '../layout';
import type { Layout } from '../layout';
import { neighbourOf, scoreToward } from '../neighbour';
import type { Direction } from '../neighbour';
import { BOUNDARIES, GRAPH, KEYS, OPTIONS, PRODUIT } from './prototype';
import type { ProtoKeys } from './prototype';

const DIRECTIONS: readonly Direction[] = ['left', 'right', 'up', 'down'];

function laidOut(walk: ProtoKeys): Layout {
    return layout(GRAPH, walk.sel, walk.viewport, OPTIONS);
}

function rectOf(found: Layout, id: string): Rect {
    const rect = found.rects.get(id);
    if (rect === undefined) {
        throw new Error(`no rect for ${id}`);
    }
    return rect;
}

type Move = { from: string; direction: Direction; theirs: null | string; ours: null | string };

function movesOf(walk: ProtoKeys): Move[] {
    const found = laidOut(walk);
    return Object.entries(walk.moves).flatMap(([from, targets]) =>
        DIRECTIONS.map((direction, index) => ({
            direction,
            from,
            ours: neighbourOf(found.rects, from, direction, GEOMETRY),
            theirs: targets[index] ?? null,
        })),
    );
}

function tied(walk: ProtoKeys, move: Move): boolean {
    if (move.ours === null || move.theirs === null) {
        return false;
    }
    const found = laidOut(walk);
    const from = rectOf(found, move.from);
    return (
        scoreToward(from, rectOf(found, move.ours), move.direction, GEOMETRY) ===
        scoreToward(from, rectOf(found, move.theirs), move.direction, GEOMETRY)
    );
}

describe('neighbour', () => {
    test('every arrow pressed on every node of eight walks moves the focus where the prototype moved it', () => {
        // Given - the prototype's arrows from every node drawn at the root, a seat, a page, a section and a leaf
        const moves = KEYS.flatMap((walk) => movesOf(walk).map((move) => ({ move, walk })));

        // When - each is asked of the ported score
        const wrong = moves
            .filter(({ move }) => move.ours !== move.theirs)
            .filter(({ move, walk }) => !tied(walk, move))
            .map(
                ({ move }) =>
                    `${move.from} ${move.direction}: ${String(move.ours)} not ${String(move.theirs)}`,
            );

        // Then - each lands on the prototype's node, or on one at exactly its score
        expect(moves).toHaveLength(368);
        expect(wrong).toStrictEqual([]);
    });

    test('a node with nothing ahead keeps the focus, and an unknown node moves nowhere', () => {
        // Given - the root at 1440, whose first seat sits at the top left
        const walk = KEYS[0];
        if (walk === undefined) {
            throw new Error('no walk dumped');
        }
        const found = laidOut(walk);

        // Then - nothing lies up or left of the first seat, and a node not drawn has no neighbour
        expect(neighbourOf(found.rects, 'S:blueprint', 'up', GEOMETRY)).toBeNull();
        expect(neighbourOf(found.rects, 'S:nowhere', 'down', GEOMETRY)).toBeNull();
    });

    test('a node a pull hid is brought back into view where the prototype brought it, from every start probed', () => {
        // Given - every node of every walk, the camera at its target and pushed far either way
        const wrong = KEYS.flatMap((walk) => {
            const frame = { geometry: GEOMETRY, layout: laidOut(walk), viewport: walk.viewport };
            return walk.visible.flatMap(({ from, id, out }) => {
                // When - the node is to be shown
                const found = cameraShowing(frame, from, id);

                // Then - the camera is where the prototype's ensureVisible put it
                return found.x === out.x && found.y === out.y
                    ? []
                    : [`${id} from ${JSON.stringify(from)}: ${JSON.stringify(found)}`];
            });
        });
        expect(wrong).toStrictEqual([]);
    });

    test('a node already in view leaves the camera untouched, the same object handed back', () => {
        // Given - Boundaries open at 1440, the camera at the origin where its seat shows
        const walk = KEYS.find(
            (one) =>
                one.sel.length === 2 && one.sel[1] === BOUNDARIES && one.viewport.width === 1440,
        );
        if (walk === undefined) {
            throw new Error('no walk with Boundaries open');
        }
        const frame = { geometry: GEOMETRY, layout: laidOut(walk), viewport: walk.viewport };
        const seat = rectOf(frame.layout, PRODUIT);
        const at = { x: 100 - seat.x, y: 100 - seat.y };

        // When - the seat is to be shown
        const found = cameraShowing(frame, at, PRODUIT);

        // Then - nothing moved
        expect(found).toBe(at);
    });
});
