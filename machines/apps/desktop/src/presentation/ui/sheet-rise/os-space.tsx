import { useLayoutEffect, useMemo, useReducer, useRef } from 'react';
import type { RefObject } from 'react';

import { Canvas } from '../canvas/canvas';
import { CANVAS_WORDS } from '../canvas/canvas.stories';
import { nodeOf, pathTo } from '../canvas/graph';
import { WIKI_BANDS, WIKI_GRAPH, WIKI_WORDS, wikiMeasure } from '../canvas/samples/wiki';
import { CollapsedBar } from './collapsed-bar';
import {
    aimAt,
    BAR_NODE_WORDS,
    BAR_WORDS,
    barPathOf,
    leafDepth,
    pageOf,
    ReadPage,
    SHEET_WORDS,
    STRIP_WORDS,
    stripOf,
} from './os-space-sample';
import type { Reading } from './os-space-sample';
import { RISE } from './rise';
import { Sheet, useWheelPast } from './sheet';
import { Strip } from './strip';

import styles from './os-space.module.css';

type Space = { path: readonly string[]; leaf: null | string; raised: boolean; reading: Reading };

type Step =
    | { kind: 'leaf'; leaf: null | string }
    | { kind: 'lower'; at?: readonly string[] | undefined }
    | { kind: 'path'; path: readonly string[] }
    | { kind: 'raise' }
    | { kind: 'raiseAt'; id: string }
    | { kind: 'read'; id: string };

function samePath(a: readonly string[], b: readonly string[]): boolean {
    return a.length === b.length && a.every((one, at) => one === b[at]);
}

// invariant: a step that changes nothing hands back the same state, and a step keeps every part it leaves alone, so
// invariant: what is handed that part (the canvas its path and leaf, the bar its reading) never renders for it.
function raisedOn(now: Space, page: null | string, at: null | string): Space {
    const kept = now.reading.page === page && now.reading.at === at;
    return now.raised && kept
        ? now
        : { ...now, raised: true, reading: kept ? now.reading : { at, page } };
}

function raiseAt(now: Space, id: string): Space {
    const to = pathTo(WIKI_GRAPH, id, leafDepth);
    const node = nodeOf(WIKI_GRAPH, id);
    const path = samePath(now.path, to) ? now.path : to;
    const leaf = node.depth >= leafDepth ? id : null;
    return raisedOn({ ...now, leaf, path }, pageOf(path), node.depth >= 3 ? id : null);
}

// value: the prototype's `lower`: the canvas opens on the page read, and down to the section read when it is past
// value: the page's first; a leaf stays when it sits in that section.
function lowered(now: Space): Space {
    const { at, page } = now.reading;
    if (page === null) {
        return { ...now, raised: false };
    }
    const node = nodeOf(WIKI_GRAPH, page);
    const section = at === null ? undefined : pathTo(WIKI_GRAPH, at, leafDepth)[2];
    const deep = section !== undefined && section !== node.kids[0];
    const to = [node.parent ?? '', page, ...(deep ? [section] : [])];
    const leaf =
        now.leaf !== null && nodeOf(WIKI_GRAPH, now.leaf).parent === to[2] ? now.leaf : null;
    return { ...now, leaf, path: samePath(now.path, to) ? now.path : to, raised: false };
}

function readOne(now: Space, id: string): Space {
    const node = nodeOf(WIKI_GRAPH, id);
    if (node.depth === 2) {
        return { ...now, path: [node.parent ?? '', id], reading: { at: null, page: id } };
    }
    return { ...now, reading: { ...now.reading, at: id } };
}

function step(now: Space, next: Step): Space {
    switch (next.kind) {
        case 'leaf': {
            return now.leaf === next.leaf ? now : { ...now, leaf: next.leaf };
        }
        case 'lower': {
            if (next.at === undefined) {
                return now.raised ? lowered(now) : now;
            }
            return { ...now, leaf: null, path: next.at, raised: false };
        }
        case 'path': {
            return { ...now, path: next.path };
        }
        case 'raise': {
            return raisedOn(now, pageOf(now.path), now.path[2] ?? null);
        }
        case 'raiseAt': {
            return raiseAt(now, next.id);
        }
        case 'read': {
            return readOne(now, next.id);
        }
    }
}

function started({ start, startRaised = false }: OsSpaceSceneProps): Space {
    return {
        leaf: null,
        path: start,
        raised: startRaised,
        reading: { at: start[2] ?? null, page: pageOf(start) },
    };
}

export type OsSpaceSceneProps = { start: readonly string[]; startRaised?: boolean };

function stepsOf(go: (next: Step) => void) {
    return {
        leaf: (leaf: null | string) => {
            go({ kind: 'leaf', leaf });
        },
        lower: (at?: readonly string[]) => {
            go({ at, kind: 'lower' });
        },
        path: (path: readonly string[]) => {
            go({ kind: 'path', path });
        },
        raise: () => {
            go({ kind: 'raise' });
        },
        raiseAt: (id: string) => {
            go({ id, kind: 'raiseAt' });
        },
        read: (id: string) => {
            go({ id, kind: 'read' });
        },
    };
}

// value: the place read brought into view each time the page rises or reads another place.
function useAim(frame: RefObject<HTMLDivElement | null>, { raised, reading }: Space): void {
    useLayoutEffect(() => {
        if (raised && frame.current !== null) {
            aimAt(frame.current, reading.at);
        }
    }, [frame, raised, reading]);
}

// value: the OS space in the gallery: the canvas of the wiki sample, its strip following the page the path shows, the
// value: sheet that reads it and the bar that folds the path while it is raised, wired by local state as W5's page
// value: will wire them, each callback one step of `step`.
export function OsSpaceScene(props: OsSpaceSceneProps) {
    const frame = useRef<HTMLDivElement | null>(null);
    const [now, go] = useReducer(step, props, started);
    const on = useMemo(() => stepsOf(go), []);
    const wheelPast = useWheelPast(on.raise);
    useAim(frame, now);
    const strip = useMemo(() => stripOf(now.path), [now.path]);
    const barPath = useMemo(() => barPathOf(now.reading), [now.reading]);

    return (
        <div className={styles.frame} ref={frame}>
            <Sheet
                bar={
                    <CollapsedBar
                        graph={WIKI_GRAPH}
                        onLower={on.lower}
                        onRead={on.read}
                        path={barPath}
                        reads={1}
                        shared={BAR_WORDS}
                        words={BAR_NODE_WORDS}
                    />
                }
                behind={
                    <Canvas
                        bands={WIKI_BANDS}
                        bottom={strip.line === undefined ? RISE.peek.title : RISE.peek.line}
                        graph={WIKI_GRAPH}
                        label="The wiki"
                        leaf={now.leaf}
                        measureText={wikiMeasure}
                        onLeaf={on.leaf}
                        onPath={on.path}
                        onRaise={on.raiseAt}
                        onWheelPast={wheelPast}
                        path={now.path}
                        shared={CANVAS_WORDS}
                        words={WIKI_WORDS}
                    />
                }
                label={strip.title}
                onLower={on.lower}
                onRaise={on.raise}
                raised={now.raised}
                strip={<Strip {...strip} onOpen={on.raise} words={STRIP_WORDS} />}
                words={SHEET_WORDS}
            >
                <ReadPage page={now.raised ? now.reading.page : pageOf(now.path)} />
            </Sheet>
        </div>
    );
}
