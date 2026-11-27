import { describe, expect, test } from 'vitest';

import { GEOMETRY } from '../geometry';
import { nodeOf, pathTo } from '../graph';
import { BOUNDARIES, GRAPH, PRODUIT, WHAT_IT_IS_NOT } from './prototype';

describe('graph', () => {
    test.each([
        ['a seat', PRODUIT, [PRODUIT]],
        ['a page', BOUNDARIES, [PRODUIT, BOUNDARIES]],
        ['a section', WHAT_IT_IS_NOT, [PRODUIT, BOUNDARIES, WHAT_IT_IS_NOT]],
        [
            'a block, which never joins the path',
            'B:product/boundaries:not-a-method',
            [PRODUIT, BOUNDARIES, WHAT_IT_IS_NOT],
        ],
    ] as const)(
        'the path to %s runs from its seat down, to its deepest non-leaf',
        (_name, id, path) => {
            // Given - the prototype's graph, whose leaves are at depth 4
            // Then - the path holds the ancestors and the node, never a leaf
            expect(pathTo(GRAPH, id, GEOMETRY.leafDepth)).toStrictEqual(path);
        },
    );

    test('a node the graph does not hold is refused by name', () => {
        // Given - an id no node has
        // Then - the lookup names it
        expect(() => nodeOf(GRAPH, 'S:nowhere')).toThrow('the graph holds no node S:nowhere');
    });
});
