import type { Measured } from '../../measure';
import type { NodeWords } from '../canvas-node';
import type { CanvasGraph, CanvasNode } from '../graph';
import graphSample from './wiki-graph.json';
import wordsSample from './wiki-words.json';

import styles from '../canvas.module.css';

// value: the wiki the owner approved, as the prototype drew it (scripts/canvas-dump.ts): its graph, the width of each
// value: label in the pill's font, the words of each node and each seat's pixel icon.

type Said = {
    count: string;
    foot: string;
    kicker: string;
    lead: string;
    line: string;
    seat: string;
    title: string;
};

const ICONS: Readonly<Record<string, readonly string[]>> = wordsSample.icons;

const SAID: Readonly<Record<string, Said>> = wordsSample.nodes;

const WIDTHS: Readonly<Record<string, number>> = graphSample.widths;

function Pixels({ rows }: { rows: readonly string[] }) {
    const columns = Math.max(...rows.map((row) => row.length));
    const grid: Measured = { '--columns': columns };
    return (
        <span aria-hidden="true" className={styles.pixels} style={grid}>
            {rows.flatMap((row, line) =>
                row
                    .padEnd(columns, '.')
                    .split('')
                    .map((cell, at) => (
                        <i
                            className={cell === '#' ? undefined : styles.off}
                            key={`${String(line)}:${String(at)}`}
                        />
                    )),
            )}
        </span>
    );
}

const SAID_KEYS = ['count', 'foot', 'kicker', 'line', 'title'] as const;

function wordsOf(node: CanvasNode): NodeWords {
    const said = SAID[node.id];
    const words: NodeWords = { name: node.label };
    if (said === undefined) {
        return words;
    }
    for (const key of SAID_KEYS) {
        if (said[key] !== '') {
            words[key] = said[key];
        }
    }
    if (node.depth === 2 && said.lead !== '') {
        words.lead = said.lead;
    }
    const rows = ICONS[said.seat];
    if (rows !== undefined) {
        words.icon = <Pixels rows={rows} />;
    }
    return words;
}

const NODES: CanvasNode[] = graphSample.nodes;

export const WIKI_GRAPH: CanvasGraph = {
    nodes: new Map(NODES.map((node) => [node.id, node])),
    roots: graphSample.roots,
};

export const WIKI_WORDS: ReadonlyMap<string, NodeWords> = new Map(
    NODES.map((node) => [node.id, wordsOf(node)]),
);

export const WIKI_BANDS: ReadonlyMap<number, string> = new Map(
    Object.entries(wordsSample.bands).map(([band, said]) => [Number(band), said]),
);

// value: the prototype's own measure of each label, so the sample places every pill where the prototype did.
export function wikiMeasure(text: string): number {
    return WIDTHS[text] ?? text.length * 7;
}
