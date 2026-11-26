import { describe, expect, test } from 'vitest';

import { faceOf } from '../faces';
import type { CanvasNode, Face } from '../graph';
import { GRAPH, stateOf, STATES, WIDE } from './prototype';
import type { ProtoState } from './prototype';

const NODES = [...GRAPH.nodes.values()];

function facesAt(sel: readonly string[]): Map<string, Face | null> {
    return new Map(NODES.map((node) => [node.id, faceOf(node, sel)]));
}

function withFace(faces: Map<string, Face | null>, face: Face, depth: number): string[] {
    return NODES.filter((node) => node.depth === depth && faces.get(node.id) === face).map(
        (node) => node.id,
    );
}

function siblingsOf(node: CanvasNode): string[] {
    return NODES.filter((other) => other.parent === node.parent && other.id !== node.id).map(
        (other) => other.id,
    );
}

function pathOf(state: ProtoState): readonly string[] {
    return state.sel;
}

describe('faces', () => {
    test.each([
        ['a seat', 'S:product', 'region'],
        ['a page', 'P:product/boundaries', 'card'],
        ['a section', 'C:product/boundaries:1', null],
        ['a block', 'B:product/boundaries:not-a-replacement', null],
    ] as const)('at the root, %s wears its face of the root', (_name, id, face) => {
        // Given - nothing open
        const node = GRAPH.nodes.get(id);

        // Then - seats are regions, pages their cards, and nothing deeper is drawn
        expect(node === undefined ? 'missing' : faceOf(node, [])).toBe(face);
    });

    test.each([
        ['the seat', 'S:product', 'open'],
        ['a seat beside it', 'S:growth', 'pill'],
        ['its page', 'P:product/boundaries', 'card'],
        ['a page of another seat', 'P:growth/positioning', null],
        ['its section', 'C:product/boundaries:1', null],
    ] as const)('with a seat open, %s wears its face', (_name, id, face) => {
        // Given - Produit open
        const node = GRAPH.nodes.get(id);

        // Then - the open one is open, its siblings pills, its children cards, the rest undrawn
        expect(node === undefined ? 'missing' : faceOf(node, ['S:product'])).toBe(face);
    });

    test.each([
        ['the seat, which is on the path', 'S:product', 'open'],
        ['a seat beside it', 'S:growth', 'dot'],
        ['the page', 'P:product/boundaries', 'open'],
        ['a page beside it', 'P:product/stack', 'pill'],
        ['a section', 'C:product/boundaries:1', 'card'],
        ['a block of a section', 'B:product/boundaries:not-a-replacement', null],
    ] as const)('with a page open, %s wears its face', (_name, id, face) => {
        // Given - Produit and Boundaries open
        const node = GRAPH.nodes.get(id);

        // Then - the path is open, the ancestors' siblings dots, the open page's siblings pills
        expect(
            node === undefined ? 'missing' : faceOf(node, ['S:product', 'P:product/boundaries']),
        ).toBe(face);
    });

    test('the open path holds exactly one open node per depth, in every state', () => {
        // Given - every state the prototype was driven through
        const wrong = STATES.flatMap((state) => {
            const faces = facesAt(pathOf(state));
            return NODES.map((node) => node.depth)
                .filter((depth, index, all) => all.indexOf(depth) === index)
                .flatMap((depth) => {
                    const open = withFace(faces, 'open', depth);
                    const want = state.sel[depth - 1] === undefined ? [] : [state.sel[depth - 1]];
                    return JSON.stringify(open) === JSON.stringify(want)
                        ? []
                        : [`depth ${depth} of ${state.id ?? 'root'}: ${open.join(',')}`];
                });
        });

        // Then - the open nodes are the path's, one per depth
        expect(wrong).toStrictEqual([]);
    });

    test('the siblings of the deepest open node are pills and those of its ancestors dots, in every state', () => {
        // Given - every state with something open
        const wrong = STATES.filter((state) => state.sel.length > 0).flatMap((state) => {
            const faces = facesAt(state.sel);
            return state.sel.flatMap((id, index) => {
                const node = GRAPH.nodes.get(id);
                const want = index === state.sel.length - 1 ? 'pill' : 'dot';
                return (node === undefined ? [] : siblingsOf(node))
                    .filter((sibling) => faces.get(sibling) !== want)
                    .map(
                        (sibling) =>
                            `${sibling} is ${String(faces.get(sibling))} in ${state.id ?? 'root'}`,
                    );
            });
        });

        // Then - none wears the other's face
        expect(wrong).toStrictEqual([]);
    });

    test('the children of the deepest open node are cards, and no other node is drawn below the path', () => {
        // Given - the section open, the deepest state of the captures
        const state = stateOf(WIDE, 'C:product/boundaries:1');
        const faces = facesAt(state.sel);

        // Then - its six blocks are cards, and the blocks of other sections are undrawn
        const cards = NODES.filter((node) => faces.get(node.id) === 'card' && node.depth === 4);
        expect(cards.map((node) => node.parent)).toStrictEqual(
            cards.map(() => 'C:product/boundaries:1'),
        );
        expect(cards).toHaveLength(6);
        expect(faces.get('B:product/boundaries:machines-refuses')).toBeNull();
    });
});
