import type { Camera, Clamped } from '../camera';
import { GEOMETRY } from '../geometry';
import type { CanvasGraph, CanvasNode, Rect } from '../graph';
import type { Guide, Viewport } from '../layout';
import type { Options } from '../place';
import graphFixture from './prototype-graph.json';
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
