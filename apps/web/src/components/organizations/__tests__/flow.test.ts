import { describe, expect, test } from 'vitest';

import { flowHeight, flowOf } from '../flow';

describe('flow', () => {
    test('each level is a row centred on the axis, linked by the reports_to each role declares', () => {
        // Given - a general over two captains, one of whom reports to him
        const { nodes, edges } = flowOf([
            { level: 1, name: 'captain-a', reports_to: 'general' },
            { level: 0, name: 'general' },
            { level: 1, name: 'captain-b' },
        ]);

        // Then - the general alone on top, the captains side by side below, one declared edge
        expect(nodes.map((node) => [node.id, node.type, node.position])).toStrictEqual([
            ['general', 'roleNode', { x: 0, y: 0 }],
            ['captain-a', 'roleNode', { x: -130, y: 200 }],
            ['captain-b', 'roleNode', { x: 130, y: 200 }],
        ]);
        expect(edges.map((edge) => edge.id)).toStrictEqual(['general->captain-a']);
    });

    test('with no reports_to declared, every role of a level is linked to every role of the next', () => {
        // Given - two leads over one worker, no role naming its superior (an empty name counts as none)
        const { nodes, edges } = flowOf([
            { level: 0, name: 'lead-a' },
            { level: 0, name: 'lead-b', reports_to: '' },
            { level: 1, name: 'worker' },
        ]);

        // Then - both leads reach the worker; only the top row lacks an inbound handle
        expect(edges.map((edge) => edge.id)).toStrictEqual(['lead-a->worker', 'lead-b->worker']);
        expect(nodes.map((node) => [node.id, node.data.isFirst, node.data.isLast])).toStrictEqual([
            ['lead-a', true, false],
            ['lead-b', true, false],
            ['worker', false, true],
        ]);
    });

    test('the canvas grows with the number of levels, from a floor of 300 pixels', () => {
        // Given - one level, then three
        // Then - one level keeps the floor, three levels need more
        expect(flowHeight([{ level: 0, name: 'solo' }])).toBe(300);
        expect(
            flowHeight([
                { level: 0, name: 'a' },
                { level: 1, name: 'b' },
                { level: 2, name: 'c' },
            ]),
        ).toBe(640);
    });
});
