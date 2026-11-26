import type { WikiFloorHead, WikiPageHead, WikiServedBlock } from '@jterrazz/station-api';
import { useState } from 'react';

import { WIKI_WORDS } from '../../words/wiki';
import { DecisionLook } from './decision';
import { DotIcon } from './dot-icon';
import { attentionOf, Fields, Findings, placeOf } from './fields';
import type { Reader } from './fields';
import { FigureLook, TableLook } from './looks';

import styles from './wiki.module.css';

function Head({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    return (
        <header className={styles.blockHead}>
            <DotIcon grids={reader.grids} name={block.Icon} size="small" />
            <span className={styles.kind}>{block.Title}</span>
            <span className={styles.word}>{block.Word}</span>
            <FloorMark floor={block.OwnFloor} />
        </header>
    );
}

export function FloorMark({ floor }: { floor: null | WikiFloorHead }) {
    if (floor === null) {
        return null;
    }
    return (
        <span className={styles.floorMark} title={floor.Question}>
            {WIKI_WORDS.floorShort}
            {floor.Index} {floor.Name}
        </span>
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
    if (block.Decision !== null) {
        return <DecisionLook block={block} decision={block.Decision} reader={reader} />;
    }
    return (
        <article
            aria-label={`${block.Title} ${block.Word}`}
            className={`${styles[look] ?? ''} ${attentionOf(block.NeedsYou)}`}
            id={placeOf(block.Address)}
        >
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
        <li
            className={`${styles.lineItem ?? ''} ${attentionOf(block.NeedsYou)}`}
            id={placeOf(block.Address)}
        >
            <button
                aria-expanded={open}
                className={styles.line}
                onClick={() => {
                    setOpen(!open);
                }}
                type="button"
            >
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.lineText}>
                    {block.Line === '' ? block.Title : block.Line}
                </span>
                <span className={styles.lineWord}>{block.Word}</span>
                {block.Decision === null || !block.Decision.Resolves ? null : (
                    <span className={styles.floorMark}>
                        {block.Decision.Number} {block.Decision.Status}
                    </span>
                )}
                <FloorMark floor={block.OwnFloor} />
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

export function Card({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    return <Whole block={block} look="card" reader={reader} />;
}

export function TileBlock({ block, reader }: { block: WikiServedBlock; reader: Reader }) {
    return <Whole block={block} look="tile" reader={reader} />;
}
