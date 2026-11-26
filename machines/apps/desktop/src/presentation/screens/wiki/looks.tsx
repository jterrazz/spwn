import type { WikiServedBlock, WikiServedFigure, WikiServedTable } from '@jterrazz/station-api';

import { SEPARATOR } from '../../../domain/separator';
import { Bar } from '../../ui/index';
import { WIKI_WORDS } from '../../words/wiki';
import { DotIcon } from './dot-icon';
import { attentionOf, FieldValue, placeOf, Ref } from './fields';
import type { Reader } from './fields';

import styles from './wiki.module.css';

function columnWord(column: string): string {
    return column.replaceAll('_', ' ');
}

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
        <section
            aria-label={`${WIKI_WORDS.table} ${block.Word}`}
            className={`${styles.look ?? ''} ${attentionOf(block.NeedsYou)}`}
            id={placeOf(block.Address)}
        >
            <header className={styles.blockHead}>
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.kind}>{WIKI_WORDS.table}</span>
                <span className={styles.tableOf}>
                    {table.Title}
                    {table.Where === '' ? null : `${SEPARATOR}${table.Where}`}
                </span>
                <span className={styles.headEnd}>
                    <Ref reader={reader} to={table.In} />
                </span>
            </header>
            <table className={styles.table}>
                <thead>
                    <tr>
                        <th>{table.Title}</th>
                        {columns.map((column) => (
                            <th key={column}>{columnWord(column)}</th>
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
                                <td data-column={columnWord(cell.Name)} key={cell.Name}>
                                    <FieldValue field={cell} reader={reader} />
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </section>
    );
}

// value: where a quad draws a part by the engine's `at`: high on the y axis alone top left, on x alone bottom right,
// value: on both top right in the accent, on neither bottom left.
const CORNER: Readonly<Record<string, string | undefined>> = {
    none: styles.atNone,
    x: styles.atX,
    xy: styles.atXY,
    y: styles.atY,
};

function Quad({ figure }: { figure: WikiServedFigure }) {
    const parts = figure.Parts ?? [];
    const placed = parts.filter((part) => CORNER[part.at ?? ''] !== undefined);
    const unplaced = parts.filter((part) => CORNER[part.at ?? ''] === undefined);
    return (
        <div className={styles.quad}>
            <span className={styles.axisY}>{figure.Y}</span>
            <div className={styles.quadField}>
                {placed.map((part) => (
                    <span
                        className={`${styles.quadPart} ${CORNER[part.at ?? ''] ?? ''}`}
                        key={part.name}
                    >
                        <i aria-hidden="true" className={styles.quadDot} />
                        {part.name}
                    </span>
                ))}
            </div>
            <span className={styles.axisX}>{figure.X}</span>
            {unplaced.length === 0 ? null : (
                <p className={styles.unplaced}>
                    {unplaced.map((part) => part.name).join(SEPARATOR)}
                    {SEPARATOR}
                    {WIKI_WORDS.unplaced}
                </p>
            )}
        </div>
    );
}

// value: a stack draws each part's count as a bar against the largest, the engine's order kept.
function Stack({ figure }: { figure: WikiServedFigure }) {
    const parts = figure.Parts ?? [];
    const most = Math.max(1, ...parts.map((part) => part.n ?? 0));
    return (
        <ol className={styles.stack}>
            {parts.map((part) => (
                <li className={styles.stackRow} key={part.name}>
                    <span className={styles.partName}>{part.name}</span>
                    <Bar label={part.name} percent={((part.n ?? 0) / most) * 100} />
                    <span className={styles.partN}>{part.n ?? 0}</span>
                </li>
            ))}
        </ol>
    );
}

function Parts({ figure }: { figure: WikiServedFigure }) {
    if (figure.Form === 'quad') {
        return <Quad figure={figure} />;
    }
    if (figure.Form === 'stack') {
        return <Stack figure={figure} />;
    }
    return (
        <ol className={styles.figureParts}>
            {(figure.Parts ?? []).map((part) => (
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
        <figure
            aria-label={`${block.Title} ${block.Word}`}
            className={`${styles.figure ?? ''} ${attentionOf(block.NeedsYou)}`}
            id={placeOf(block.Address)}
        >
            <header className={styles.blockHead}>
                <DotIcon grids={reader.grids} name={block.Icon} size="small" />
                <span className={styles.kind}>{block.Title}</span>
                <span className={styles.word}>{figure.Form}</span>
                <span className={styles.headEnd}>
                    <Ref reader={reader} to={figure.Shows} />
                </span>
            </header>
            <div className={styles.figureBody}>
                <Parts figure={figure} />
                <figcaption className={styles.caption}>{figure.Caption}</figcaption>
            </div>
        </figure>
    );
}
