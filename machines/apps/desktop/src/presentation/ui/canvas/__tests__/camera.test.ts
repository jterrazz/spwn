import { describe, expect, test } from 'vitest';

import { cameraClamp, cameraTarget } from '../camera';
import type { Frame } from '../camera';
import { GEOMETRY } from '../geometry';
import { layout } from '../layout';
import type { Viewport } from '../layout';
import {
    BOUNDARIES,
    GRAPH,
    NARROW,
    OPTIONS,
    stateOf,
    STATES,
    WHAT_IT_IS_NOT,
    WIDE,
} from './prototype';
import type { ProtoState } from './prototype';

function frameOf(state: ProtoState, viewport: Viewport = state.viewport): Frame {
    return { geometry: GEOMETRY, layout: layout(GRAPH, state.sel, viewport, OPTIONS), viewport };
}

describe('camera', () => {
    test('every state the prototype was driven through puts the camera where the prototype did', () => {
        // Given - the 47 states, each with the camera target the prototype computed
        const wrong = STATES.flatMap((state) => {
            // When - the target is asked of the same layout
            const target = cameraTarget(frameOf(state), state.sel);

            // Then - it is the dump's
            return target.x === state.target.x && target.y === state.target.y
                ? []
                : [`${state.id ?? 'root'} at ${state.viewport.width}: ${JSON.stringify(target)}`];
        });
        expect(wrong).toStrictEqual([]);
    });

    test('a root wider than the window is shown from its left edge, 48 px in', () => {
        // Given - the root at 1100, where the floors are 1240 px wide
        const state = stateOf(NARROW, null);

        // When - the camera is targeted
        const target = cameraTarget(frameOf(state), state.sel);

        // Then - the box starts 48 px from the left, and its height is centred
        expect(target).toStrictEqual({ x: 48, y: 48 });
    });

    test('a root that fits is centred on both axes', () => {
        // Given - the root at 1440, 1240 by 644 in a canvas of 1440 by 740
        const state = stateOf(WIDE, null);

        // When - the camera is targeted
        const target = cameraTarget(frameOf(state), state.sel);

        // Then - the free room is split evenly
        expect(target).toStrictEqual({ x: 100, y: 48 });
    });

    test('the open node whose parent fits is centred through that parent', () => {
        // Given - Boundaries open: its seat is 1100 by 318, which fits
        const state = stateOf(WIDE, BOUNDARIES);
        const frame = frameOf(state);
        const seat = frame.layout.rects.get('S:product');

        // When - the camera is targeted
        const target = cameraTarget(frame, state.sel);

        // Then - the seat's centre is the canvas's centre
        expect([
            target.x + (seat?.x ?? 0) + (seat?.w ?? 0) / 2,
            target.y + (seat?.y ?? 0) + (seat?.h ?? 0) / 2,
        ]).toStrictEqual([state.viewport.width / 2, state.viewport.height / 2]);
    });

    test('an open node too tall for the canvas is shown from 70 px under the top', () => {
        // Given - the section (290 tall) open in a canvas only 300 px tall, where neither it nor its page (430) fits
        const state = stateOf(WIDE, WHAT_IT_IS_NOT);
        const frame = frameOf(state, { height: 300, width: 1440 });
        const section = frame.layout.rects.get(WHAT_IT_IS_NOT);

        // When - the camera is targeted
        const target = cameraTarget(frame, state.sel);

        // Then - the section's top is 70 px down, and it is still centred across
        expect([
            target.y + (section?.y ?? 0),
            target.x + (section?.x ?? 0) + (section?.w ?? 0) / 2,
        ]).toStrictEqual([70, 720]);
    });

    test('every pull the prototype was probed with is held to the same bounds, with the same edges reported', () => {
        // Given - four pulls per state, from two starting positions
        const wrong = STATES.flatMap((state) =>
            state.clamps.flatMap(({ from, out, to }) => {
                // When - the pull is clamped
                const found = cameraClamp(frameOf(state), from, to);

                // Then - the clamped point and both edge flags are the prototype's
                return JSON.stringify([found.x, found.y, found.atBottom, found.atTop]) ===
                    JSON.stringify([out.x, out.y, out.atBottom, out.atTop])
                    ? []
                    : [`${state.id ?? 'root'} ${JSON.stringify(from)} to ${JSON.stringify(to)}`];
            }),
        );
        expect(wrong).toStrictEqual([]);
    });

    test('a pull far past the content keeps 120 px of it in view, and a pull inside is left alone', () => {
        // Given - the root at 1440, pulled from the origin
        const frame = frameOf(stateOf(WIDE, null));
        const origin = { x: 0, y: 0 };

        // When - it is pulled far left and up, far right and down, and a little
        const lost = cameraClamp(frame, origin, { x: -9000, y: -9000 });
        const held = cameraClamp(frame, origin, { x: 9000, y: 9000 });
        const free = cameraClamp(frame, origin, { x: 5, y: -3 });

        // Then - 120 px of content stay in view at the far end, and the small pull passes
        expect([lost.x, lost.y]).toStrictEqual([0, 740 - 120 - 644]);
        expect([held.x, held.y]).toStrictEqual([120, 120]);
        expect([free.x, free.y]).toStrictEqual([5, -3]);
        expect([lost.atBottom, held.atTop, free.atTop, free.atBottom]).toStrictEqual([
            true,
            true,
            false,
            false,
        ]);
    });
});
