import type { ReactNode } from 'react';

import { Bone, Panel, RowSkeleton, Tile, VerdictSkeleton } from '../ui/index';
import type { RowSkeletonProps } from '../ui/row-skeleton';
import type { VerdictSkeletonProps } from '../ui/verdict';
import { readingWord } from '../words/words';

import styles from './skeleton.module.css';

// value: what a page draws while a read has no answer yet: the same panels, rows and heights its answer will hold,
// value: busy for assistive tech, which hears the reading sentence once while the drawing itself is hidden from it.
export function Skeleton({ what, children }: { what: string; children: ReactNode }) {
    return (
        <div aria-busy="true" className={styles.region}>
            <p className={styles.said} role="status">
                {readingWord(what)}
            </p>
            {children}
        </div>
    );
}

// value: an answer kept on screen while the read that replaces it has not answered: drawn as it was, dimmed and
// value: busy, so the page keeps its place and says it is about to change.
export function Placeheld({ held, children }: { held: boolean; children: ReactNode }) {
    return (
        <div aria-busy={held} className={held ? styles.held : styles.region}>
            {children}
        </div>
    );
}

export type SkeletonHeadProps = VerdictSkeletonProps & { what: string };

// value: a page whose read draws only its verdict before the answer: the verdict's bones, busy.
export function SkeletonHead({ what, ...verdict }: SkeletonHeadProps) {
    return (
        <Skeleton what={what}>
            <VerdictSkeleton {...verdict} />
        </Skeleton>
    );
}

const ROWS = ['first', 'second', 'third', 'fourth', 'fifth', 'sixth'] as const;

export type SkeletonRowsProps = RowSkeletonProps & { count?: 1 | 2 | 3 | 4 | 5 | 6 };

// invariant: one bone row unless the page knows its count, so a first answer only ever grows the page.
export function SkeletonRows({ count = 1, ...row }: SkeletonRowsProps) {
    return ROWS.slice(0, count).map((place) => <RowSkeleton key={place} {...row} />);
}

export type SkeletonPanelProps = SkeletonRowsProps & { title: string; head?: ReactNode };

// value: a panel whose title the window knows before the answer, its rows drawn as bones.
export function SkeletonPanel({ title, head, ...rows }: SkeletonPanelProps) {
    return (
        <div aria-hidden="true">
            <Panel title={title}>
                {head}
                <SkeletonRows {...rows} />
            </Panel>
        </div>
    );
}

export function SkeletonTile({ label }: { label: string }) {
    return <Tile foot={<Bone width="long" />} label={label} value={<Bone width="short" />} />;
}
