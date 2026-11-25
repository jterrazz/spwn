import type {
    WikiParagraph,
    WikiServedBlock,
    WikiServedGroup,
    WikiServedItem,
} from '@jterrazz/station-api';
import { useState } from 'react';

import { SEPARATOR } from '../../../domain/separator';
import { countOf } from '../../words/engine-words';
import { WIKI_FOLDED_WORDS, WIKI_WORDS } from '../../words/wiki';
import { Cards, Lines, TileBlock } from './block';
import { DotIcon } from './dot-icon';
import type { Reader } from './fields';
import { Prose } from './prose';

import styles from './wiki.module.css';

type Run =
    | { kind: 'card' | 'line'; blocks: WikiServedBlock[] }
    | { kind: 'counted'; count: number }
    | { kind: 'group'; group: WikiServedGroup }
    | { kind: 'prose'; paragraphs: WikiParagraph[] }
    | { kind: 'tile'; block: WikiServedBlock };

function blockRun(runs: Run[], block: WikiServedBlock): void {
    const before = runs.at(-1);
    const { Shown } = block;
    if (Shown === 'tile') {
        runs.push({ block, kind: 'tile' });
    } else if (Shown === 'counted' && before?.kind === 'counted') {
        before.count += 1;
    } else if (Shown === 'counted') {
        runs.push({ count: 1, kind: 'counted' });
    } else if ((Shown === 'card' || Shown === 'line') && before?.kind === Shown) {
        before.blocks.push(block);
    } else if (Shown === 'card' || Shown === 'line') {
        runs.push({ blocks: [block], kind: Shown });
    }
}

// invariant: the engine decided how each block shows; the window only sets the blocks it shows alike side by side,
// invariant: so cards share one grid, lines one list, and the blocks counted one sentence.
function runsOf(items: readonly WikiServedItem[]): Run[] {
    const runs: Run[] = [];
    for (const item of items) {
        if (item.Prose !== null) {
            runs.push({ kind: 'prose', paragraphs: item.Prose });
        } else if (item.Group !== null) {
            runs.push({ group: item.Group, kind: 'group' });
        } else if (item.Block !== null) {
            blockRun(runs, item.Block);
        }
    }
    return runs;
}

function Counted({ count }: { count: number }) {
    return <p className={styles.counted}>{countOf('quiet-block', count)}</p>;
}

function RunView({ run, reader }: { run: Run; reader: Reader }) {
    switch (run.kind) {
        case 'prose': {
            return <Prose paragraphs={run.paragraphs} />;
        }
        case 'group': {
            return <Group group={run.group} reader={reader} />;
        }
        case 'tile': {
            return <TileBlock block={run.block} reader={reader} />;
        }
        case 'card': {
            return <Cards blocks={run.blocks} reader={reader} />;
        }
        case 'line': {
            return <Lines blocks={run.blocks} reader={reader} />;
        }
        case 'counted': {
            return <Counted count={run.count} />;
        }
    }
}

function keyOf(run: Run, at: number): string {
    if (run.kind === 'group') {
        return `group-${run.group.ID}`;
    }
    if (run.kind === 'tile') {
        return `tile-${run.block.Address}`;
    }
    if (run.kind === 'card' || run.kind === 'line') {
        return `${run.kind}-${run.blocks[0]?.Address ?? String(at)}`;
    }
    return `${run.kind}-${String(at)}`;
}

export function Items({ items, reader }: { items: readonly WikiServedItem[]; reader: Reader }) {
    return runsOf(items).map((run, at) => (
        <RunView key={keyOf(run, at)} reader={reader} run={run} />
    ));
}

function blocksOf(items: readonly WikiServedItem[]): WikiServedBlock[] {
    return items.flatMap((item) => (item.Block === null ? [] : [item.Block]));
}

function foldedWord(group: WikiServedGroup, seat: string): string {
    return group.Folded === 'seat'
        ? `${WIKI_FOLDED_WORDS.seat} ${seat}`
        : WIKI_FOLDED_WORDS[group.Folded];
}

function Folded({
    group,
    reader,
    onOpen,
}: {
    group: WikiServedGroup;
    reader: Reader;
    onOpen: () => void;
}) {
    const blocks = blocksOf(group.Items ?? []);
    return (
        <div className={styles.folded}>
            {group.Lead === '' ? null : <p className={styles.groupLead}>{group.Lead}</p>}
            <div className={styles.chips}>
                {blocks.map((block) => (
                    <button
                        className={styles.chip}
                        key={block.Address}
                        onClick={onOpen}
                        type="button"
                    >
                        <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                        {block.Word}
                    </button>
                ))}
            </div>
        </div>
    );
}

function Group({ group, reader }: { group: WikiServedGroup; reader: Reader }) {
    const folds = group.Folded !== '';
    const [open, setOpen] = useState(!folds);
    const meta = [
        group.Intent,
        countOf('block', group.Blocks),
        folds ? foldedWord(group, reader.seat) : '',
    ]
        .filter((part) => part !== '')
        .join(SEPARATOR);
    return (
        <section aria-label={group.Title} className={styles.group}>
            <header className={styles.groupHead}>
                <span className={styles.groupIcon}>
                    <DotIcon grids={reader.grids} name={group.Intent} size="mid" />
                </span>
                <span className={styles.groupTitles}>
                    <h3 className={styles.groupTitle}>{group.Title}</h3>
                    <span className={styles.groupMeta}>{meta}</span>
                </span>
                {folds ? (
                    <button
                        aria-expanded={open}
                        className={styles.groupToggle}
                        onClick={() => {
                            setOpen(!open);
                        }}
                        type="button"
                    >
                        {open ? WIKI_WORDS.fold : WIKI_WORDS.open}
                    </button>
                ) : null}
            </header>
            {open ? (
                <div className={styles.groupBody}>
                    <Items items={group.Items ?? []} reader={reader} />
                </div>
            ) : (
                <Folded
                    group={group}
                    onOpen={() => {
                        setOpen(true);
                    }}
                    reader={reader}
                />
            )}
        </section>
    );
}
