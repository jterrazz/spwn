import { Fragment, memo } from 'react';
import type { ReactElement } from 'react';

import type { NodeWords } from '../canvas/canvas-node';
import { nodeOf } from '../canvas/graph';
import type { CanvasGraph } from '../canvas/graph';
import { CrumbDots } from '../crumb-dots';
import type { CrumbDot } from '../crumb-dots';
import { Icon } from '../icon';

import styles from './collapsed-bar.module.css';

// value: what the bar says of its own: its landmark's name, the home button's name, the way back to the canvas, its
// value: key as ARIA names it and as its keycap shows it, a count of needs, and the name of the dot drawn current
// value: (its own tab stop, as the prototype's, so it is never named as the pill before it).
export type CollapsedWords = {
    label: string;
    home: string;
    canvas: string;
    key: string;
    keycap: string;
    needs: (count: number) => string;
    current: (name: string) => string;
};

export type CollapsedBarProps = {
    graph: CanvasGraph;
    path: readonly string[];
    reads: number;
    words: ReadonlyMap<string, NodeWords>;
    shared: CollapsedWords;
    onLower: (path?: readonly string[]) => void;
    onRead: (id: string) => void;
};

const NO_WORDS: NodeWords = { name: '' };

function named(graph: CanvasGraph, id: string, name: string, shared: CollapsedWords): string {
    const { needs } = nodeOf(graph, id);
    return needs > 0 ? `${name}, ${shared.needs(needs)}` : name;
}

// value: the prototype's siblings of a crumb (`drawCbar`): at the first level the other roots, deeper every kid of
// value: the parent with the crumb itself drawn current; none when the crumb is alone.
function siblingsOf(
    { graph, path, shared, words }: CollapsedBarProps,
    at: number,
): readonly CrumbDot[] {
    const id = path[at] ?? '';
    const parent = at === 0 ? null : nodeOf(graph, id).parent;
    const kin =
        parent === null ? graph.roots.filter((one) => one !== id) : nodeOf(graph, parent).kids;
    if (at > 0 && kin.length < 2) {
        return [];
    }
    return kin.map((one) => {
        const name = named(graph, one, (words.get(one) ?? NO_WORDS).name, shared);
        return {
            id: one,
            name: one === id ? shared.current(name) : name,
            on: one === id,
            you: nodeOf(graph, one).needs > 0,
        };
    });
}

// value: the navigation folded over a raised page: home, then each node of the read path as a pill with its siblings
// value: as dots, then the way back to the canvas. Home, a pill and a dot above the page read (`reads`, the path's
// value: index of the page) lower the sheet onto that place of the canvas; a dot at or under the page keeps it raised
// value: and reads that node; "Canvas" lowers it where the page was read, as Escape does.
function crumbOf(props: CollapsedBarProps, id: string, at: number): ReactElement {
    const { graph, onLower, onRead, path, reads, shared, words } = props;
    const said = words.get(id) ?? NO_WORDS;
    const dots = siblingsOf(props, at);
    return (
        <Fragment key={id}>
            <span aria-hidden="true" className={styles.slash}>
                /
            </span>
            <button
                aria-label={named(graph, id, said.name, shared)}
                className={styles.pill}
                onClick={() => {
                    onLower(path.slice(0, at + 1));
                }}
                type="button"
            >
                {said.icon}
                <span>{said.name}</span>
            </button>
            {dots.length > 0 ? (
                <CrumbDots
                    dots={dots}
                    onPick={(dot) => {
                        if (at >= reads) {
                            onRead(dot);
                        } else {
                            onLower([...path.slice(0, at), dot]);
                        }
                    }}
                />
            ) : null}
        </Fragment>
    );
}

function CollapsedBarOf(props: CollapsedBarProps) {
    const { onLower, path, shared } = props;
    return (
        <nav aria-label={shared.label} className={styles.bar}>
            <button
                aria-label={shared.home}
                className={styles.home}
                onClick={() => {
                    onLower([]);
                }}
                title={shared.home}
                type="button"
            >
                <Icon name="home" />
            </button>
            {path.map((id, at) => crumbOf(props, id, at))}
            <button
                aria-keyshortcuts={shared.key}
                className={styles.back}
                onClick={() => {
                    onLower();
                }}
                type="button"
            >
                <Icon name="up" />
                {shared.canvas}
                <kbd aria-hidden="true">{shared.keycap}</kbd>
            </button>
        </nav>
    );
}

export const CollapsedBar = memo(CollapsedBarOf);
