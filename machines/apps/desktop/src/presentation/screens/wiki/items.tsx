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
import { Card, Lines, TileBlock } from './block';
import { DotIcon } from './dot-icon';
import { Findings } from './fields';
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
// invariant: so cards share one row, lines one list, and the blocks counted one sentence.
export function runsOf(items: readonly WikiServedItem[]): Run[] {
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

// invariant: a light section holds no wide tile and no group, so it can share a shelf with the next one.
export function isLight(items: readonly WikiServedItem[]): boolean {
    return runsOf(items).every((run) => run.kind !== 'group' && run.kind !== 'tile');
}

type Cell =
    | { kind: 'block'; block: WikiServedBlock; look: 'card' | 'tile' }
    | { kind: 'group'; group: WikiServedGroup }
    | { kind: 'wide'; run: Run };

// value: a cell the page sets side by side with its neighbours: a card, a sub-page, a group served folded.
function isSmall(cell: Cell): boolean {
    if (cell.kind === 'group') {
        return cell.group.Folded !== '';
    }
    return cell.kind === 'block' && (cell.look === 'card' || cell.block.Sub !== null);
}

function cellsOf(runs: readonly Run[]): Cell[] {
    return runs.flatMap((run): Cell[] => {
        if (run.kind === 'card') {
            return run.blocks.map((block) => ({ block, kind: 'block', look: 'card' }));
        }
        if (run.kind === 'tile') {
            return [{ block: run.block, kind: 'block', look: 'tile' }];
        }
        if (run.kind === 'group') {
            return [{ group: run.group, kind: 'group' }];
        }
        return [{ kind: 'wide', run }];
    });
}

// value: the span of a small cell by how many share its row: one, two or three.
const IN_ROW = [styles.whole, styles.whole, styles.half, styles.third] as const;
const ROW = 3;

// invariant: small cells side by side share their row equally, three at most to a row; a small cell alone takes the row.
function spansOf(cells: readonly Cell[]): (string | undefined)[] {
    const spans: (string | undefined)[] = [];
    let start = 0;
    for (const [at, cell] of cells.entries()) {
        if (isSmall(cell)) {
            let end = at;
            for (
                let next = cells[at + 1];
                next !== undefined && isSmall(next);
                next = cells[end + 1]
            ) {
                end += 1;
            }
            const rowStart = start + Math.floor((at - start) / ROW) * ROW;
            spans.push(IN_ROW[Math.min(ROW, end - rowStart + 1)]);
        } else {
            spans.push(styles.whole);
            start = at + 1;
        }
    }
    return spans;
}

function WideView({ run, reader }: { run: Run; reader: Reader }) {
    if (run.kind === 'prose') {
        return (
            <div className={styles.proseCell}>
                <Prose paragraphs={run.paragraphs} />
            </div>
        );
    }
    if (run.kind === 'line') {
        return (
            <div className={styles.whole}>
                <Lines blocks={run.blocks} reader={reader} />
            </div>
        );
    }
    if (run.kind === 'counted') {
        return <p className={styles.counted}>{countOf('quiet-block', run.count)}</p>;
    }
    return null;
}

function CellView({
    cell,
    reader,
    span,
}: {
    cell: Cell;
    reader: Reader;
    span: string | undefined;
}) {
    if (cell.kind === 'wide') {
        return <WideView reader={reader} run={cell.run} />;
    }
    if (cell.kind === 'group') {
        return <Group group={cell.group} reader={reader} span={span} />;
    }
    return (
        <div className={span}>
            {cell.look === 'card' ? (
                <Card block={cell.block} reader={reader} />
            ) : (
                <TileBlock block={cell.block} reader={reader} />
            )}
        </div>
    );
}

function keyOf(cell: Cell, at: number): string {
    if (cell.kind === 'group') {
        return `group-${cell.group.ID}`;
    }
    if (cell.kind === 'block') {
        return `block-${cell.block.Address}`;
    }
    return `${cell.run.kind}-${String(at)}`;
}

export function Bento({ items, reader }: { items: readonly WikiServedItem[]; reader: Reader }) {
    const cells = cellsOf(runsOf(items));
    const spans = spansOf(cells);
    return (
        <div className={styles.bento}>
            {cells.map((cell, at) => (
                <CellView cell={cell} key={keyOf(cell, at)} reader={reader} span={spans[at]} />
            ))}
        </div>
    );
}

function blocksOf(items: readonly WikiServedItem[]): WikiServedBlock[] {
    return items.flatMap((item) => (item.Block === null ? [] : [item.Block]));
}

function foldedWord(group: WikiServedGroup, seat: string): string {
    return group.Folded === 'seat'
        ? `${WIKI_FOLDED_WORDS.seat} ${seat}`
        : WIKI_FOLDED_WORDS[group.Folded];
}

function GroupHead({
    group,
    reader,
    toggle,
}: {
    group: WikiServedGroup;
    reader: Reader;
    toggle: undefined | { open: boolean; flip: () => void };
}) {
    const meta = [
        group.Intent,
        countOf('block', group.Blocks),
        group.Folded === '' ? '' : foldedWord(group, reader.seat),
    ]
        .filter((part) => part !== '')
        .join(SEPARATOR);
    return (
        <header className={styles.groupHead}>
            <span className={styles.groupIcon}>
                <DotIcon grids={reader.grids} name={group.Intent} size="mid" />
            </span>
            <span className={styles.groupTitles}>
                <h3 className={styles.groupTitle}>{group.Title}</h3>
                <span className={styles.groupMeta}>{meta}</span>
            </span>
            {toggle === undefined ? null : (
                <button
                    aria-expanded={toggle.open}
                    className={styles.groupToggle}
                    onClick={toggle.flip}
                    type="button"
                >
                    {toggle.open ? WIKI_WORDS.fold : WIKI_WORDS.open}
                </button>
            )}
        </header>
    );
}

function Group({
    group,
    reader,
    span,
}: {
    group: WikiServedGroup;
    reader: Reader;
    span: string | undefined;
}) {
    const folds = group.Folded !== '';
    const [open, setOpen] = useState(!folds);
    const toggle = folds
        ? {
              flip: () => {
                  setOpen(!open);
              },
              open,
          }
        : undefined;
    if (!open) {
        const blocks = blocksOf(group.Items ?? []);
        return (
            <section aria-label={group.Title} className={`${span ?? ''} ${styles.foldedGroup}`}>
                <GroupHead group={group} reader={reader} toggle={toggle} />
                {group.Lead === '' ? null : <p className={styles.groupLead}>{group.Lead}</p>}
                <div className={styles.chips}>
                    {blocks.map((block) => (
                        <button
                            className={styles.chip}
                            key={block.Address}
                            onClick={() => {
                                setOpen(true);
                            }}
                            type="button"
                        >
                            <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                            {block.Word}
                        </button>
                    ))}
                </div>
                <Findings findings={blocks.flatMap((block) => block.Finding ?? [])} />
            </section>
        );
    }
    return (
        <section aria-label={group.Title} className={`${styles.whole} ${styles.group}`}>
            <GroupHead group={group} reader={reader} toggle={toggle} />
            <Bento items={group.Items ?? []} reader={reader} />
        </section>
    );
}
