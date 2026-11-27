import { memo, useMemo, useRef } from 'react';

import type { Measured } from '../measure';
import { CanvasNodeView } from './canvas-node';
import type { CanvasWords, NodeWords } from './canvas-node';
import { GEOMETRY } from './geometry';
import { handlersOf, useWheel } from './gestures';
import type { Actions, WheelPast } from './gestures';
import { nodeOf, pathTo } from './graph';
import type { CanvasGraph, Face, Rect } from './graph';
import { useMotion } from './motion';
import { CanvasRig } from './rig';
import { useFlip, useRove, useSize, useStage } from './stage';
import type { Staged } from './stage';

import styles from './canvas.module.css';

export type CanvasProps = {
    graph: CanvasGraph;
    path: readonly string[];
    leaf: null | string;
    label: string;
    words: ReadonlyMap<string, NodeWords>;
    shared: CanvasWords;
    bands?: ReadonlyMap<number, string>;
    bottom?: number;
    measureText?: (text: string) => number;
    onPath: (path: readonly string[]) => void;
    onLeaf: (id: null | string) => void;
    onRaise?: (id: string) => void;
    onWheelPast?: WheelPast;
};

const NO_BANDS: ReadonlyMap<number, string> = new Map();

const NO_WORDS: NodeWords = { name: '' };

const { leafDepth } = GEOMETRY;

type Acting = Pick<CanvasProps, 'graph' | 'leaf' | 'onLeaf' | 'onPath' | 'onRaise' | 'path'> & {
    faceOf: (id: string) => Face | undefined;
};

// value: the prototype's `activate` and `up`: a leaf is marked, a second click on it raises it; an open node closes
// value: to its parent; any other node opens with its path; up drops the leaf first, then the deepest open node.
function actionsOf({ faceOf, graph, leaf, onLeaf, onPath, onRaise, path }: Acting): Actions {
    const raise = (id: string) => {
        onRaise?.(id);
    };
    const activate = (id: string, by: 'click' | 'key') => {
        const node = nodeOf(graph, id);
        if (node.depth >= leafDepth) {
            if (leaf === id && by === 'click') {
                raise(id);
            } else {
                onLeaf(id);
            }
            return;
        }
        if (leaf !== null) {
            onLeaf(null);
        }
        onPath(
            faceOf(id) === 'open' ? path.slice(0, node.depth - 1) : pathTo(graph, id, leafDepth),
        );
    };
    return {
        activate,
        enter: (id) => {
            if (faceOf(id) === 'open' || (leaf === id && nodeOf(graph, id).depth >= leafDepth)) {
                raise(id);
            } else {
                activate(id, 'key');
            }
        },
        raise,
        up: () => {
            if (leaf !== null) {
                onLeaf(null);
            } else if (path.length > 0) {
                onPath(path.slice(0, -1));
            }
        },
    };
}

// value: the nodes drawn, the leaving among them, deepest last so an open node's kids sit over its ground, and in
// value: the graph's order within a depth, so a change only inserts or removes and never moves a node holding focus.
const ORDERS = new WeakMap<CanvasGraph, ReadonlyMap<string, number>>();

function orderOf(graph: CanvasGraph): ReadonlyMap<string, number> {
    const known = ORDERS.get(graph);
    if (known !== undefined) {
        return known;
    }
    const order = new Map([...graph.nodes.keys()].map((id, at) => [id, at]));
    ORDERS.set(graph, order);
    return order;
}

function drawnIds(graph: CanvasGraph, staged: Staged): string[] {
    if (staged.frame.viewport.width === 0) {
        return [];
    }
    const order = orderOf(graph);
    const { rects } = staged.frame.layout;
    const leaving = [...staged.leaving.keys()].filter((id) => !rects.has(id));
    return [...rects.keys(), ...leaving].toSorted(
        (a, b) =>
            nodeOf(graph, a).depth - nodeOf(graph, b).depth ||
            (order.get(a) ?? 0) - (order.get(b) ?? 0),
    );
}

const PITCH = 22;

// value: the dot grid, as far past the content as the prototype draws it, aligned to its pitch.
function gridOf(box: Rect): Measured {
    return {
        '--gh': `${String(box.h + 2000)}px`,
        '--gw': `${String(box.w + 2800)}px`,
        '--gx': `${String(Math.floor((box.x - 1400) / PITCH) * PITCH)}px`,
        '--gy': `${String(Math.floor((box.y - 1000) / PITCH) * PITCH)}px`,
    };
}

function placed({ w, x, y }: { x: number; y: number; w: number }): Measured {
    return { '--w': `${String(w)}px`, '--x': `${String(x)}px`, '--y': `${String(y)}px` };
}

// value: what the live region says when a node opens: its name and its count, "Produit, 3 pages".
function saidOf(words: ReadonlyMap<string, NodeWords>, opened: string | undefined): string {
    const said = opened === undefined ? undefined : words.get(opened);
    return said === undefined
        ? ''
        : [said.name, said.count].filter((one) => one !== undefined).join(', ');
}

// value: a canvas of nodes over a dot grid, each drawn by its face (region, open, card, pill, dot) at the place the
// value: pure layout gives it, under one camera. It is controlled: the open path and the leaf live with the caller,
// value: handed back through `onPath` and `onLeaf`, and Space, Read or a second click on the leaf calls `onRaise`;
// value: a wheel down past the camera's bottom is handed to `onWheelPast`.
function CanvasOf(props: CanvasProps) {
    const {
        bands = NO_BANDS,
        bottom = 0,
        graph,
        label,
        leaf,
        measureText,
        onWheelPast,
        path,
        shared,
        words,
    } = props;
    const motion = useMotion();
    const root = useRef<HTMLDivElement | null>(null);
    const world = useRef<HTMLDivElement | null>(null);
    // hazard: the rig holds the camera and the node elements for the canvas's life; React may drop a memo's value,
    // hazard: which would only cost the camera's place and the next layout's motion, never a wrong drawing.
    const rig = useMemo(() => new CanvasRig(world), []);
    const staged = useStage({ bottom, graph, measureText, motion, path, size: useSize(root) });
    const { frame } = staged;
    useFlip({ graph, motion, path, rig, staged });
    useRove(root, frame, leaf ?? path.at(-1) ?? null);
    useWheel(root, rig, frame, onWheelPast);
    const actions = actionsOf({ ...props, faceOf: (id) => frame.layout.rects.get(id)?.face });
    const handlers = handlersOf({ actions, frame, motion, rig });

    return (
        <div aria-label={label} className={styles.canvas} ref={root} role="group" {...handlers}>
            <div className={styles.world} ref={world}>
                <div className={styles.grid} style={gridOf(frame.layout.box)} />
                {frame.layout.guides.map((guide) => (
                    <div className={styles.guide} key={guide.band} style={placed(guide)}>
                        <span>{bands.get(guide.band) ?? ''}</span>
                    </div>
                ))}
                {drawnIds(graph, staged).map((id) => (
                    <CanvasNodeView
                        face={
                            frame.layout.rects.get(id)?.face ??
                            staged.leaving.get(id)?.face ??
                            'dot'
                        }
                        key={id}
                        leaf={leaf === id}
                        leafDepth={leafDepth}
                        motion={motion}
                        node={nodeOf(graph, id)}
                        onElement={rig.hold}
                        shared={shared}
                        words={words.get(id) ?? NO_WORDS}
                    />
                ))}
            </div>
            <p aria-live="polite" className={styles.said}>
                {saidOf(words, path.at(-1))}
            </p>
        </div>
    );
}

// invariant: memoised, so what holds it (the sheet over it) renders without rendering the canvas.
export const Canvas = memo(CanvasOf);
