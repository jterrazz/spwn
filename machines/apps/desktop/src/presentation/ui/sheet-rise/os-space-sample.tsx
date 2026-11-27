import { memo } from 'react';

import type { NodeWords } from '../canvas/canvas-node';
import { CANVAS_WORDS } from '../canvas/canvas.stories';
import { GEOMETRY } from '../canvas/geometry';
import { nodeOf, pathTo } from '../canvas/graph';
import { WIKI_GRAPH, WIKI_WORDS } from '../canvas/samples/wiki';
import { Icon } from '../icon';
import type { CollapsedWords } from './collapsed-bar';
import type { SheetWords } from './sheet';
import type { StripProps, StripWords } from './strip';

import styles from './os-space.module.css';

export const STRIP_WORDS: StripWords = {
    hint: 'scroll ↓ or',
    name: (title) => `Open ${title}`,
    needs: CANVAS_WORDS.needs,
    open: 'Open',
};

export const BAR_WORDS: CollapsedWords = {
    canvas: 'Canvas',
    current: (name) => `${name}, current`,
    home: 'The canvas, at the root',
    key: 'Escape',
    keycap: 'esc',
    label: 'Where this page sits',
    needs: CANVAS_WORDS.needs,
};

export const SHEET_WORDS: SheetWords = { grab: 'Raise or lower the page' };

export const { leafDepth } = GEOMETRY;

const NO_WORDS: NodeWords = { name: '' };

export function wordsOf(id: string): NodeWords {
    return WIKI_WORDS.get(id) ?? NO_WORDS;
}

// value: the bar's words: the canvas's, a page drawn by the prototype's book rather than its seat's pixels.
export const BAR_NODE_WORDS: ReadonlyMap<string, NodeWords> = new Map(
    [...WIKI_WORDS].map(([id, words]) => [
        id,
        nodeOf(WIKI_GRAPH, id).depth === 2 ? { ...words, icon: <Icon name="book" /> } : words,
    ]),
);

const PAGES = [...WIKI_GRAPH.nodes.values()].filter((node) => node.depth === 2);

const NEEDS = WIKI_GRAPH.roots.reduce((sum, id) => sum + nodeOf(WIKI_GRAPH, id).needs, 0);

const WAITING = PAGES.filter((page) => page.needs > 0).length;

// value: the page the open path shows: the open page, else the open seat's first, else none (the wiki's own home).
export function pageOf(path: readonly string[]): null | string {
    const [seat, page] = path;
    if (page !== undefined) {
        return page;
    }
    return seat === undefined ? null : (nodeOf(WIKI_GRAPH, seat).kids[0] ?? null);
}

export type Reading = { page: null | string; at: null | string };

// value: the path the bar folds: the read page's seat and the page, then the section read, else the page's first.
export function barPathOf({ at, page }: Reading): readonly string[] {
    if (page === null) {
        return [];
    }
    const node = nodeOf(WIKI_GRAPH, page);
    const section = at === null ? node.kids[0] : pathTo(WIKI_GRAPH, at, leafDepth)[2];
    return [node.parent ?? '', page, ...(section === undefined ? [] : [section])];
}

export type Shown = Omit<StripProps, 'onOpen' | 'words'>;

const HOME: Shown = {
    icon: <Icon name="home" />,
    line: `Every page of this wiki, by the seat that reads it, and the ${String(NEEDS)} findings on ${String(WAITING)} pages that wait for you.`,
    meta: ['Wiki', `${String(PAGES.length)} pages in ${String(WIKI_GRAPH.roots.length)} seats`],
    needs: NEEDS,
    title: 'Wiki',
};

// value: the strip's words for the page the path shows, from the sample's own titles and lines; the reading time
// value: the engine will serve is not in the sample, so the page's count of sections stands in its place.
export function stripOf(path: readonly string[]): Shown {
    const page = pageOf(path);
    if (page === null) {
        return HOME;
    }
    const node = nodeOf(WIKI_GRAPH, page);
    const words = wordsOf(page);
    const section = path[2];
    const meta = [wordsOf(node.parent ?? '').name, words.kicker ?? '', words.foot ?? ''].filter(
        (part) => part !== '',
    );
    return {
        icon: words.icon,
        line: words.line,
        meta: section === undefined ? meta : [...meta, `at ${wordsOf(section).name}`],
        needs: node.needs,
        title: words.title ?? words.name,
    };
}

function HomePage() {
    return (
        <article className={styles.page}>
            <h1>{HOME.title}</h1>
            <p className={styles.lead}>{HOME.line}</p>
            {WIKI_GRAPH.roots.map((seat) => (
                <section className={styles.section} data-anchor={seat} key={seat}>
                    <h2>{wordsOf(seat).name}</h2>
                    {nodeOf(WIKI_GRAPH, seat).kids.map((one) => (
                        <p key={one}>
                            <b>{wordsOf(one).title ?? wordsOf(one).name}</b> {wordsOf(one).line}
                        </p>
                    ))}
                </section>
            ))}
        </article>
    );
}

// value: a stand-in for the page W3 draws: the page's title and line, then each section's heading and each of its
// value: blocks' title and line, all the sample's own words; the wiki's home lists every seat's pages.
function ReadPageOf({ page }: { page: null | string }) {
    if (page === null) {
        return <HomePage />;
    }
    const node = nodeOf(WIKI_GRAPH, page);
    const words = wordsOf(page);
    return (
        <article className={styles.page}>
            <p className={styles.kicker}>
                {[wordsOf(node.parent ?? '').name, words.kicker].join(' · ')}
            </p>
            <h1>{words.title ?? words.name}</h1>
            <p className={styles.lead}>{words.line}</p>
            {node.kids.map((section) => (
                <section className={styles.section} data-anchor={section} key={section}>
                    <h2>{wordsOf(section).title ?? wordsOf(section).name}</h2>
                    {nodeOf(WIKI_GRAPH, section).kids.map((block) => (
                        <p data-anchor={block} key={block}>
                            <b>{wordsOf(block).name}</b> {wordsOf(block).line}
                        </p>
                    ))}
                </section>
            ))}
        </article>
    );
}

export const ReadPage = memo(ReadPageOf);

// value: the prototype's `openSheet` and `scrollToEl`: the page read from its top, or the place read brought 28 px
// value: under the page's top.
const AIM_MARGIN = 28;

export function aimAt(frame: HTMLElement, id: null | string): void {
    const scroller = frame.querySelector<HTMLElement>('[data-scroll]');
    if (scroller === null) {
        return;
    }
    scroller.scrollTop = 0;
    const target = id === null ? null : frame.querySelector<HTMLElement>(`[data-anchor="${id}"]`);
    if (target === null) {
        return;
    }
    const by = target.getBoundingClientRect().top - scroller.getBoundingClientRect().top;
    scroller.scrollTop = by - AIM_MARGIN;
}
