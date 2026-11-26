import type { Camera, Clamped } from '../camera';
import { GEOMETRY } from '../geometry';
import type { CanvasGraph, CanvasNode, Rect } from '../graph';
import type { Guide, Viewport } from '../layout';
import type { Options } from '../place';
import graphFixture from './prototype-graph.json';
import keysFixture from './prototype-keys.json';
import statesFixture from './prototype-states.json';

export type ProtoRect = Rect & { face: string };

export type ProtoState = {
    id: null | string;
    sel: string[];
    viewport: Viewport;
    rects: Record<string, ProtoRect>;
    guides: Guide[];
    box: Rect;
    target: Camera;
    clamps: { from: Camera; to: Camera; out: Clamped }[];
};

const NODES: CanvasNode[] = graphFixture.nodes;

const WIDTHS: Record<string, number> = graphFixture.widths;

export const GRAPH: CanvasGraph = {
    nodes: new Map(NODES.map((node) => [node.id, node])),
    roots: graphFixture.roots,
};

export const OPTIONS: Options = {
    geometry: GEOMETRY,
    measureText: (text) => {
        const width = WIDTHS[text];
        if (width === undefined) {
            throw new Error(`the prototype measured no label "${text}"`);
        }
        return width;
    },
};

export const WIDE: ProtoState[] = statesFixture.wide;

export const NARROW: ProtoState[] = statesFixture.narrow;

export const STATES: ProtoState[] = [...WIDE, ...NARROW];

// value: one walk of the prototype's keys: for each node drawn under `sel`, where each arrow (left, right, up,
// value: down) moved the focus, null where it stayed, and where `ensureVisible` took the camera from three starts.
export type ProtoKeys = {
    sel: string[];
    leaf: null | string;
    viewport: Viewport;
    moves: Record<string, (null | string)[]>;
    visible: { id: string; from: Camera; out: Camera }[];
};

export const KEYS: ProtoKeys[] = [...keysFixture.wide, ...keysFixture.narrow];

export const PRODUIT = 'S:product';

export const BOUNDARIES = 'P:product/boundaries';

export const WHAT_IT_IS_NOT = 'C:product/boundaries:1';

export function stateOf(states: readonly ProtoState[], id: null | string): ProtoState {
    const found = states.find((state) => state.id === id);
    if (found === undefined) {
        throw new Error(`the prototype dump holds no state for ${id}`);
    }
    return found;
}

export function facesOf(
    rects: Iterable<readonly [string, { face: string }]>,
): Record<string, string> {
    return Object.fromEntries([...rects].map(([id, rect]) => [id, rect.face]));
}
