import type { WikiServedBlock, WikiServedFigure, WikiServedTable } from '@jterrazz/station-api';

import { WIKI_WORDS } from '../../words/wiki';
import { DotIcon } from './dot-icon';
import { FieldValue, Ref } from './fields';
import type { Reader } from './fields';

import styles from './wiki.module.css';

export function TableLook({
    block,
    table,
    reader,
}: {
    block: WikiServedBlock;
    table: WikiServedTable;
    reader: Reader;
}) {
    const columns = table.Columns ?? [];
    return (
        <section aria-label={`${WIKI_WORDS.table} ${block.Word}`} className={styles.look}>
            <header className={styles.blockHead}>
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.kind}>{WIKI_WORDS.table}</span>
                <span className={styles.word}>{table.Of}</span>
                {table.Where === '' ? null : (
                    <span className={styles.where}>
                        {WIKI_WORDS.where} {table.Where}
                    </span>
                )}
                <span className={styles.headEnd}>
                    <Ref reader={reader} to={table.In} />
                </span>
            </header>
            <div className={styles.scroller}>
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>{table.Title}</th>
                            {columns.map((column) => (
                                <th key={column}>{column.replaceAll('_', ' ')}</th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {(table.Rows ?? []).map((row) => (
                            <tr key={row.Address}>
                                <th className={styles.rowHead} scope="row">
                                    <DotIcon grids={reader.grids} name={row.Icon} size="small" />
                                    {row.Word}
                                </th>
                                {(row.Cells ?? []).map((cell) => (
                                    <td key={cell.Name}>
                                        <FieldValue field={cell} reader={reader} />
                                    </td>
                                ))}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </section>
    );
}

function Parts({ figure }: { figure: WikiServedFigure }) {
    const parts = figure.Parts ?? [];
    if (figure.Form === 'quad') {
        return (
            <div className={styles.quad}>
                <span className={styles.axisY}>{figure.Y}</span>
                <div className={styles.quadField}>
                    {parts.map((part) => (
                        <span className={styles.quadPart} key={part.name}>
                            {part.name}
                            {part.at === undefined ? null : <small> {part.at}</small>}
                        </span>
                    ))}
                </div>
                <span className={styles.axisX}>{figure.X}</span>
            </div>
        );
    }
    return (
        <ol className={styles.figureParts}>
            {parts.map((part) => (
                <li key={part.name}>
                    <span className={styles.partName}>{part.name}</span>
                    {part.n === undefined ? null : <span className={styles.partN}>{part.n}</span>}
                    {part.to === undefined ? null : (
                        <span className={styles.partTo}>{part.to}</span>
                    )}
                </li>
            ))}
        </ol>
    );
}

export function FigureLook({
    block,
    figure,
    reader,
}: {
    block: WikiServedBlock;
    figure: WikiServedFigure;
    reader: Reader;
}) {
    return (
        <figure aria-label={`${block.Title} ${block.Word}`} className={styles.figure}>
            <header className={styles.blockHead}>
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.kind}>{block.Title}</span>
                <span className={styles.word}>{figure.Form}</span>
                <span className={styles.headEnd}>
                    <Ref reader={reader} to={figure.Shows} />
                </span>
            </header>
            <Parts figure={figure} />
            <figcaption className={styles.caption}>{figure.Caption}</figcaption>
        </figure>
    );
}
