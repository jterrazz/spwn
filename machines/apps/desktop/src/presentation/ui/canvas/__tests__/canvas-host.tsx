import { Profiler, useState } from 'react';

import { Canvas } from '../canvas';
import { CANVAS_WORDS } from '../canvas.stories';
import { GEOMETRY } from '../geometry';
import { layout } from '../layout';
import type { Layout } from '../layout';
import { FLEET_BANDS, FLEET_SAMPLE } from '../samples/fleet';
import { WIKI_BANDS, WIKI_GRAPH, WIKI_WORDS, wikiMeasure } from '../samples/wiki';

import styles from './canvas-host.module.css';

// value: the desk the specs draw the canvas in, sized by canvas-host.module.css.
export const DESK = { height: 700, width: 1100 };

type HostProps = {
    sample: 'fleet' | 'wiki';
    start?: readonly string[];
    onRender?: () => void;
    onPath?: (path: readonly string[]) => void;
    onRaise?: (id: string) => void;
};

const nothing = (): void => undefined;

const ROOT: readonly string[] = [];

const fleetMeasure = (text: string): number => text.length * 7;

// value: a page holding the canvas: the path and the leaf in its state, each new path also told to `onPath`.
export function Host({
    onPath = nothing,
    onRaise = nothing,
    onRender = nothing,
    sample,
    start = ROOT,
}: HostProps) {
    const [path, setPath] = useState(start);
    const [leaf, setLeaf] = useState<null | string>(null);
    const wiki = sample === 'wiki';
    return (
        <div className={styles.desk}>
            <Profiler id="canvas" onRender={onRender}>
                <Canvas
                    bands={wiki ? WIKI_BANDS : FLEET_BANDS}
                    graph={wiki ? WIKI_GRAPH : FLEET_SAMPLE.graph}
                    label={wiki ? 'The wiki' : 'The fleet'}
                    leaf={leaf}
                    measureText={wiki ? wikiMeasure : fleetMeasure}
                    onLeaf={setLeaf}
                    onPath={(next) => {
                        onPath(next);
                        setPath(next);
                    }}
                    onRaise={onRaise}
                    path={path}
                    shared={CANVAS_WORDS}
                    words={wiki ? WIKI_WORDS : FLEET_SAMPLE.words}
                />
            </Profiler>
        </div>
    );
}

export function wikiLayout(path: readonly string[]): Layout {
    return layout(WIKI_GRAPH, path, DESK, { geometry: GEOMETRY, measureText: wikiMeasure });
}

// value: each node's box as drawn, in the world's coordinates: its client rect less the world's origin.
export function drawnRects(): Map<string, { x: number; y: number; w: number; h: number }> {
    const world = document.querySelector('[data-node]')?.parentElement;
    const origin = world?.getBoundingClientRect() ?? new DOMRect();
    return new Map(
        [...document.querySelectorAll<HTMLElement>('[data-node]')]
            .filter((one) => !one.inert)
            .map((one) => {
                const box = one.getBoundingClientRect();
                return [
                    one.dataset.node ?? '',
                    {
                        h: Math.round(box.height),
                        w: Math.round(box.width),
                        x: Math.round(box.left - origin.left),
                        y: Math.round(box.top - origin.top),
                    },
                ];
            }),
    );
}

// value: every animation of the document played to its end, so what is drawn is where the motion lands.
export async function settled(): Promise<void> {
    for (const running of document.getAnimations()) {
        running.finish();
    }
    await new Promise((resolve) => {
        requestAnimationFrame(resolve);
    });
}

// value: prefers-reduced-motion, answered as `reduce` by the page's matchMedia while `on`.
export function reducedMotion(on: boolean): () => void {
    const real = globalThis.matchMedia.bind(globalThis);
    globalThis.matchMedia = (query: string): MediaQueryList => {
        if (!query.includes('reduced-motion')) {
            return real(query);
        }
        return new Proxy(real(query), {
            get: (target, key): unknown => {
                if (key === 'matches') {
                    return on;
                }
                const value: unknown = Reflect.get(target, key);
                if (typeof value === 'function') {
                    const bound: unknown = Function.prototype.bind.call(value, target);
                    return bound;
                }
                return value;
            },
        });
    };
    return () => {
        globalThis.matchMedia = real;
    };
}
