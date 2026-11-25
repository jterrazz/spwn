import type { WikiPageHead, WikiServedBlock } from '@jterrazz/station-api';
import { useState } from 'react';

import { DotIcon } from './dot-icon';
import { Fields, Findings } from './fields';
import type { Reader } from './fields';
import { FigureLook, TableLook } from './looks';

import styles from './wiki.module.css';

function Head({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    return (
        <header className={styles.blockHead}>
            <DotIcon grids={reader.grids} name={block.Icon} size="small" />
            <span className={styles.kind}>{block.Title}</span>
            <span className={styles.word}>{block.Word}</span>
        </header>
    );
}

function SubPage({ sub, reader }: { sub: WikiPageHead; reader: Reader }) {
    return (
        <button
            className={styles.sub}
            onClick={() => {
                reader.open(sub.ID);
            }}
            type="button"
        >
            <DotIcon grids={reader.grids} name={sub.Icon === '' ? 'page' : sub.Icon} size="mid" />
            <span className={styles.subWord}>{sub.Word}</span>
            <span className={styles.subLine}>{sub.Line}</span>
        </button>
    );
}

function Whole({
    block,
    reader,
    look,
}: {
    block: WikiServedBlock;
    reader: Reader;
    look: 'card' | 'tile';
}) {
    if (block.Table !== null) {
        return <TableLook block={block} reader={reader} table={block.Table} />;
    }
    if (block.Figure !== null) {
        return <FigureLook block={block} figure={block.Figure} reader={reader} />;
    }
    if (block.Sub !== null) {
        return <SubPage reader={reader} sub={block.Sub} />;
    }
    return (
        <article aria-label={`${block.Title} ${block.Word}`} className={styles[look]}>
            <Head block={block} reader={reader} />
            <p className={look === 'tile' ? styles.tileLine : styles.cardLine}>{block.Line}</p>
            <Fields fields={block.Fields} line={block.Line} reader={reader} />
            <Findings findings={block.Finding} />
        </article>
    );
}

function Line({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    const [open, setOpen] = useState(false);
    return (
        <li className={styles.lineItem}>
            <button
                aria-expanded={open}
                className={styles.line}
                onClick={() => {
                    setOpen(!open);
                }}
                type="button"
            >
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.lineWord}>{block.Word}</span>
                <span className={styles.lineText}>
                    {block.Line === '' ? block.Title : block.Line}
                </span>
            </button>
            {open ? (
                <div className={styles.lineOpen}>
                    <Fields fields={block.Fields} line={block.Line} reader={reader} />
                </div>
            ) : null}
            <Findings findings={block.Finding} />
        </li>
    );
}

export function Lines({ blocks, reader }: { blocks: readonly WikiServedBlock[]; reader: Reader }) {
    return (
        <ul className={styles.lines}>
            {blocks.map((block) => (
                <Line block={block} key={block.Address} reader={reader} />
            ))}
        </ul>
    );
}

export function Cards({ blocks, reader }: { blocks: readonly WikiServedBlock[]; reader: Reader }) {
    return (
        <div className={styles.cards}>
            {blocks.map((block) => (
                <Whole block={block} key={block.Address} look="card" reader={reader} />
            ))}
        </div>
    );
}

export function TileBlock({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    return <Whole block={block} look="tile" reader={reader} />;
}
