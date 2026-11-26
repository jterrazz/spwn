import { useEffect, useLayoutEffect, useMemo, useState } from 'react';
import type { RefObject } from 'react';

import type { Frame } from './camera';
import { flip, leave, seat } from './flip';
import type { Drawn, Scene } from './flip';
import { GEOMETRY } from './geometry';
import type { CanvasGraph, Placed } from './graph';
import { layout } from './layout';
import type { Layout } from './layout';
import { curvesOf, MOTION } from './motion';
import type { Options } from './place';
import { rove } from './rig';
import type { CanvasRig } from './rig';

type Size = { width: number; height: number };

type Shown = { layout: Layout; leaving: ReadonlyMap<string, Placed>; width: number };

export type Staged = { frame: Frame; leaving: ReadonlyMap<string, Placed> };

type Staging = {
    graph: CanvasGraph;
    path: readonly string[];
    size: Size;
    bottom: number;
    measureText: ((text: string) => number) | undefined;
    motion: boolean;
};

function pillMeasure(): (text: string) => number {
    const context = document.createElement('canvas').getContext('2d');
    const family = getComputedStyle(document.documentElement).getPropertyValue('--font-sans');
    return (text) => {
        if (context === null) {
            return text.length * 7;
        }
        context.font = `500 13px ${family === '' ? 'sans-serif' : family}`;
        return context.measureText(text).width;
    };
}

// value: the nodes `after` drops that were drawn, kept for their exit; nothing is drawn before the canvas has a size.
function leavingOf(before: Shown, after: Layout, motion: boolean): ReadonlyMap<string, Placed> {
    if (!motion || before.width === 0) {
        return new Map();
    }
    return new Map(
        [...before.leaving, ...before.layout.rects].filter(([id]) => !after.rects.has(id)),
    );
}

function sizeOf(element: HTMLElement): Size {
    return { height: element.clientHeight, width: element.clientWidth };
}

export function useSize(root: RefObject<HTMLElement | null>): Size {
    const [size, setSize] = useState<Size>({ height: 0, width: 0 });
    useLayoutEffect(() => {
        const element = root.current;
        if (element === null) {
            return;
        }
        // invariant: an observer reports the size it starts from, so the first layout waits for no other read.
        const observer = new ResizeObserver(() => {
            const next = sizeOf(element);
            setSize((was) => (was.width === next.width && was.height === next.height ? was : next));
        });
        observer.observe(element);
        return () => {
            observer.disconnect();
        };
    }, [root]);
    return size;
}

// value: the roving tab stop, moved after each layout and each leaf change (rig.ts `rove`).
export function useRove(
    root: RefObject<HTMLElement | null>,
    frame: Frame,
    want: null | string,
): void {
    useLayoutEffect(() => {
        if (root.current !== null) {
            rove(root.current, want, frame.layout.rects);
        }
    }, [frame, root, want]);
}

// value: the layout of the open path in the canvas's size, and the nodes it dropped, held through their exit.
export function useStage({ bottom, graph, measureText, motion, path, size }: Staging): Staged {
    const options = useMemo<Options>(
        () => ({ geometry: GEOMETRY, measureText: measureText ?? pillMeasure() }),
        [measureText],
    );
    const viewport = useMemo(
        () => ({ height: Math.max(0, size.height - bottom), width: size.width }),
        [bottom, size],
    );
    const now = useMemo(
        () => layout(graph, path, viewport, options),
        [graph, path, viewport, options],
    );
    const [shown, setShown] = useState<Shown>({ layout: now, leaving: new Map(), width: 0 });
    if (shown.layout !== now) {
        setShown({ layout: now, leaving: leavingOf(shown, now, motion), width: viewport.width });
    }
    useEffect(() => {
        if (shown.leaving.size === 0) {
            return;
        }
        const timer = setTimeout(() => {
            setShown((was) => ({ ...was, leaving: new Map() }));
        }, MOTION.exit);
        return () => {
            clearTimeout(timer);
        };
    }, [shown.leaving]);
    const frame = useMemo<Frame>(
        () => ({ geometry: GEOMETRY, layout: now, viewport }),
        [now, viewport],
    );
    return { frame, leaving: shown.leaving };
}

type Moving = {
    rig: CanvasRig;
    staged: Staged;
    graph: CanvasGraph;
    path: readonly string[];
    motion: boolean;
};

function drawnOf(
    elements: ReadonlyMap<string, HTMLElement>,
    rects: ReadonlyMap<string, Placed>,
): Drawn[] {
    return [...rects].flatMap(([id, rect]) => {
        const element = elements.get(id);
        return element === undefined ? [] : [{ element, rect }];
    });
}

// value: a node drawn again before its exit ended comes back whole: reachable, and with its exit cancelled.
function revived(drawn: readonly Drawn[]): void {
    for (const { element } of drawn) {
        if (element.inert) {
            element.inert = false;
            for (const running of element.getAnimations()) {
                running.cancel();
            }
        }
    }
}

function exits(
    scene: Scene,
    elements: ReadonlyMap<string, HTMLElement>,
    leaving: ReadonlyMap<string, Placed>,
): void {
    for (const [id, rect] of leaving) {
        const element = elements.get(id);
        if (element !== undefined && !element.inert) {
            element.inert = true;
            leave(scene, { element, rect });
        }
    }
}

// invariant: the FLIP is measured once per layout and from the layout, never from the DOM: each node is seated at its
// invariant: new rect, then the transform and opacity animations carry it from its old one; the camera follows.
export function useFlip({ graph, motion, path, rig, staged }: Moving): void {
    const { frame, leaving } = staged;
    useLayoutEffect(() => {
        const { elements, world } = rig;
        if (world === null || frame.viewport.width === 0) {
            return;
        }
        const drawn = drawnOf(elements, frame.layout.rects);
        revived(drawn);
        const old = rig.remembered(frame.layout.rects);
        const animated = motion && old !== null;
        const scene = {
            after: frame.layout.rects,
            before: old ?? new Map(),
            curves: curvesOf(world),
            graph,
        };
        if (animated) {
            flip(scene, drawn);
        } else {
            for (const one of drawn) {
                seat(one.element, one.rect);
            }
        }
        exits(scene, elements, leaving);
        rig.settle(frame, path, animated);
    }, [frame, graph, leaving, motion, path, rig]);
}
