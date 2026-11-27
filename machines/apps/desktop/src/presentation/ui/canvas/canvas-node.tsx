import { memo, useCallback, useState } from 'react';
import type { ReactNode } from 'react';

import { Icon } from '../icon';
import type { Face, CanvasNode as Node } from './graph';

import styles from './canvas.module.css';

// value: what a node says on each face, handed by the caller, so the canvas assumes no wiki: `name` is the word a
// value: pill, a head and the accessible name carry; a card adds `kicker` over its `title` (the name by default),
// value: a `line` and a `foot` under its marks; a head adds `count`, and with `lead` the head is a page's, large,
// value: its title over that line. `icon` is drawn before the name where the face has room. A node given `read`
// value: can be read: open, it draws the Read button, named so, which hands it to the reader.
export type NodeWords = {
    name: string;
    title?: string;
    kicker?: string;
    line?: string;
    foot?: string;
    count?: string;
    lead?: string;
    icon?: ReactNode;
    read?: string;
};

// value: the words every node shares: what the close hint and the Read button say, and the accessible name of a
// value: count of needs.
export type CanvasWords = { close: string; read: string; needs: (count: number) => string };

type NodeProps = {
    node: Node;
    words: NodeWords;
    face: Face;
    leaf: boolean;
    leafDepth: number;
    shared: CanvasWords;
    motion: boolean;
    onElement: (id: string, element: HTMLDivElement | null) => void;
};

const CARD_MARKS = 10;

const PILL_MARKS = 11;

function Marks({ marks, most }: { marks: readonly boolean[]; most: number }) {
    if (marks.length === 0) {
        return null;
    }
    const shown = marks.slice(0, most);
    return (
        <span aria-hidden="true" className={styles.marks}>
            {shown.map((you, index) => (
                <i className={you ? styles.you : undefined} key={index} />
            ))}
            {marks.length > shown.length ? <i className={styles.more} /> : null}
        </span>
    );
}

function You({ count, shared }: { count: number; shared: CanvasWords }) {
    return count > 0 ? (
        <i aria-label={shared.needs(count)} className={styles.mark} role="img" />
    ) : null;
}

type Drawn = { node: Node; words: NodeWords; shared: CanvasWords; leafDepth: number };

function nameOf({ node, shared, words }: Drawn): string {
    return node.needs > 0 ? `${words.name}, ${shared.needs(node.needs)}` : words.name;
}

function head(drawn: Drawn, open: boolean): ReactNode {
    const { node, shared, words } = drawn;
    if (open && words.lead !== undefined) {
        return (
            <>
                <span className={styles.title}>
                    {words.icon}
                    <span>{words.title ?? words.name}</span>
                    <You count={node.needs} shared={shared} />
                </span>
                <span className={styles.lead}>{words.lead}</span>
            </>
        );
    }
    return (
        <>
            {words.icon}
            <span className={styles.label}>{words.name}</span>
            {words.count === undefined ? null : <span className={styles.count}>{words.count}</span>}
            <You count={node.needs} shared={shared} />
            {open ? (
                <span className={styles.close}>
                    <Icon name="up" />
                    {shared.close}
                </span>
            ) : null}
        </>
    );
}

function card({ leafDepth, node, shared, words }: Drawn): ReactNode {
    const foot = words.foot !== undefined || node.marks.length > 0;
    return (
        <>
            <span className={styles.top}>
                {words.icon}
                {words.kicker === undefined ? null : <span>{words.kicker}</span>}
                {node.needs > 0 ? (
                    <span className={styles.waits}>
                        <You count={node.needs} shared={shared} />
                        {node.depth < leafDepth ? node.needs : null}
                    </span>
                ) : null}
            </span>
            <b>{words.title ?? words.name}</b>
            {words.line === undefined ? null : (
                <span className={styles.line} data-lines={foot ? 1 : 2}>
                    {words.line}
                </span>
            )}
            {foot ? (
                <span className={styles.foot}>
                    <Marks marks={node.marks} most={CARD_MARKS} />
                    {words.foot}
                </span>
            ) : null}
        </>
    );
}

function pill({ leafDepth, node, shared, words }: Drawn): ReactNode {
    return (
        <>
            {words.icon}
            <span className={styles.word}>{words.name}</span>
            <Marks marks={node.marks} most={PILL_MARKS} />
            {node.depth >= leafDepth ? <You count={node.needs} shared={shared} /> : null}
        </>
    );
}

// value: a face's drawing, the same for the live face and its ghost; the ghost is a copy no one can reach.
function body(drawn: Drawn, face: Face): ReactNode {
    if (face === 'region' || face === 'open') {
        return head(drawn, face === 'open');
    }
    if (face === 'card') {
        return card(drawn);
    }
    if (face === 'pill' || drawn.node.depth === 1) {
        return face === 'pill' ? (
            pill(drawn)
        ) : (
            <>
                {drawn.words.icon}
                <Marks marks={drawn.node.marks} most={CARD_MARKS} />
            </>
        );
    }
    return null;
}

const KINDS: Readonly<Record<Face, string>> = {
    card: 'card',
    dot: 'dot',
    open: 'head',
    pill: 'pill',
    region: 'head',
};

function kindOf(face: Face, drawn: Drawn): string {
    if (face === 'open' && drawn.words.lead !== undefined) {
        return 'lead';
    }
    if (face === 'dot' && drawn.node.depth === 1) {
        return 'chip';
    }
    return KINDS[face];
}

// value: an open node's Read button, for a node its words name as one to read; the canvas's click hands it on.
function readOf({ node, shared, words }: Drawn): ReactNode {
    if (words.read === undefined) {
        return null;
    }
    return (
        <button aria-label={words.read} className={styles.read} data-read={node.id} type="button">
            <Icon name="book" />
            {shared.read}
        </button>
    );
}

function useElement(id: string, onElement: NodeProps['onElement']) {
    return useCallback(
        (one: HTMLDivElement | null) => {
            onElement(id, one);
        },
        [id, onElement],
    );
}

// invariant: a node renders when its face or its leaf mark changes, never when it only moves: its place is written
// invariant: by the canvas into its custom properties (flip.ts `seat`), so opening a node renders only the nodes
// invariant: whose face changed, the born and the leaf.
function CanvasNodeOf({
    face,
    leaf,
    leafDepth,
    motion,
    node,
    onElement,
    shared,
    words,
}: NodeProps) {
    const [seen, setSeen] = useState(face);
    const [ghost, setGhost] = useState<Face | null>(null);
    if (seen !== face) {
        setSeen(face);
        setGhost(motion ? seen : null);
    }
    const element = useElement(node.id, onElement);
    const drawn = { leafDepth, node, shared, words };
    const kind = kindOf(face, drawn);
    const hasKids = node.kids.length > 0 && node.depth < leafDepth;
    return (
        <div
            className={styles.node}
            data-depth={Math.min(node.depth, leafDepth)}
            data-face={face}
            data-leaf={leaf ? '' : undefined}
            data-node={node.id}
            ref={element}
        >
            <div className={styles.ground} data-part="ground" />
            <div className={styles.body} data-part="body">
                <button
                    aria-current={leaf ? 'true' : undefined}
                    aria-expanded={hasKids ? face === 'open' : undefined}
                    aria-label={nameOf(drawn)}
                    className={styles.hit}
                    data-kind={kind}
                    data-n={node.id}
                    data-you={node.needs > 0 ? '' : undefined}
                    title={face === 'dot' ? words.name : undefined}
                    type="button"
                >
                    {body(drawn, face)}
                </button>
                {face === 'open' ? readOf(drawn) : null}
            </div>
            {ghost === null ? null : (
                <div aria-hidden="true" className={styles.ghost} data-part="ghost" inert>
                    <span className={styles.hit} data-kind={kindOf(ghost, drawn)}>
                        {body(drawn, ghost)}
                    </span>
                </div>
            )}
        </div>
    );
}

export const CanvasNodeView = memo(CanvasNodeOf);
