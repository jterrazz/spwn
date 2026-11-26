import { useState } from 'react';

import type { Story } from '../story';
import { Canvas } from './canvas';
import type { CanvasProps } from './canvas';
import type { CanvasWords } from './canvas-node';
import { FLEET_BANDS, FLEET_SAMPLE } from './samples/fleet';
import { WIKI_BANDS, WIKI_GRAPH, WIKI_WORDS, wikiMeasure } from './samples/wiki';

export type CanvasSampleProps = Omit<CanvasProps, 'leaf' | 'onLeaf' | 'onPath' | 'path'> & {
    start: readonly string[];
    startLeaf?: string;
};

// value: a canvas as a page holds it: the open path and the leaf in the page's state, so the sample opens and
// value: closes as the canvas asks.
export function CanvasSample({ start, startLeaf, ...canvas }: CanvasSampleProps) {
    const [path, setPath] = useState(start);
    const [leaf, setLeaf] = useState<null | string>(startLeaf ?? null);
    return <Canvas {...canvas} leaf={leaf} onLeaf={setLeaf} onPath={setPath} path={path} />;
}

export const CANVAS_WORDS: CanvasWords = {
    close: 'close',
    needs: (count) => `${String(count)} wait${count > 1 ? '' : 's'} for you`,
};

const FLEET: Omit<CanvasSampleProps, 'start'> = {
    bands: FLEET_BANDS,
    graph: FLEET_SAMPLE.graph,
    label: 'The fleet',
    shared: CANVAS_WORDS,
    words: FLEET_SAMPLE.words,
};

const WIKI: Omit<CanvasSampleProps, 'start'> = {
    bands: WIKI_BANDS,
    graph: WIKI_GRAPH,
    label: 'The wiki',
    measureText: wikiMeasure,
    shared: CANVAS_WORDS,
    words: WIKI_WORDS,
};

export const CANVAS_STORIES: readonly Story<CanvasSampleProps>[] = [
    { name: 'fleet', props: { ...FLEET, start: [] } },
    {
        name: 'fleet, a machine open',
        props: { ...FLEET, start: ['mac:MacBook Pro', 'mac:MacBook Pro/macbook-pro'] },
    },
    { name: 'wiki', props: { ...WIKI, start: [] } },
    { name: 'wiki, a seat open', props: { ...WIKI, start: ['S:product'] } },
    { name: 'wiki, a page open', props: { ...WIKI, start: ['S:product', 'P:product/boundaries'] } },
    {
        name: 'wiki, a section open',
        props: { ...WIKI, start: ['S:product', 'P:product/boundaries', 'C:product/boundaries:1'] },
    },
];
