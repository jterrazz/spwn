import type { WikiParagraph, WikiRun } from '@jterrazz/station-api';

import styles from './wiki.module.css';

function Run({ run }: { run: WikiRun }) {
    if (run.Code === true) {
        return <code className={styles.code}>{run.Text}</code>;
    }
    if (run.Strong === true) {
        return <strong>{run.Text}</strong>;
    }
    if ((run.Link ?? '') !== '') {
        return (
            <span className={styles.link} title={run.Link}>
                {run.Text}
            </span>
        );
    }
    return run.Text;
}

export function Runs({ runs }: { runs: null | readonly WikiRun[] }) {
    return (runs ?? []).map((run, at) => <Run key={at} run={run} />);
}

type Rows = { head: undefined | WikiParagraph; rows: WikiParagraph[] };

function cellsOf(paragraph: WikiParagraph) {
    return paragraph.Cells ?? [];
}

function ProseTable({ head, rows }: Rows) {
    return (
        <div className={styles.scroller}>
            <table className={styles.proseTable}>
                {head === undefined ? null : (
                    <thead>
                        <tr>
                            {cellsOf(head).map((cell, at) => (
                                <th key={at}>
                                    <Runs runs={cell} />
                                </th>
                            ))}
                        </tr>
                    </thead>
                )}
                <tbody>
                    {rows.map((row, at) => (
                        <tr key={at}>
                            {cellsOf(row).map((cell, column) => (
                                <td key={column}>
                                    <Runs runs={cell} />
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
        </div>
    );
}

type Part =
    | (Rows & { kind: 'table' })
    | { kind: 'items'; items: WikiParagraph[] }
    | { kind: 'one'; one: WikiParagraph };

function joined(parts: Part[], paragraph: WikiParagraph): boolean {
    const before = parts.at(-1);
    if (paragraph.Form === 'row' && before?.kind === 'table') {
        before.rows.push(paragraph);
        return true;
    }
    if (paragraph.Form === 'item' && before?.kind === 'items') {
        before.items.push(paragraph);
        return true;
    }
    return false;
}

function opened(paragraph: WikiParagraph): Part {
    if (paragraph.Form === 'head') {
        return { head: paragraph, kind: 'table', rows: [] };
    }
    if (paragraph.Form === 'row') {
        return { head: undefined, kind: 'table', rows: [paragraph] };
    }
    if (paragraph.Form === 'item') {
        return { items: [paragraph], kind: 'items' };
    }
    return { kind: 'one', one: paragraph };
}

// invariant: the engine serves prose as paragraphs of one form each; the window only gathers the rows of one table
// invariant: and the items of one list, written side by side, to draw them as one table and one list.
function partsOf(paragraphs: readonly WikiParagraph[]): Part[] {
    const parts: Part[] = [];
    for (const paragraph of paragraphs) {
        if (!joined(parts, paragraph)) {
            parts.push(opened(paragraph));
        }
    }
    return parts;
}

function Drawn({ part }: { part: Part }) {
    if (part.kind === 'table') {
        return <ProseTable head={part.head} rows={part.rows} />;
    }
    if (part.kind === 'items') {
        return (
            <ul className={styles.list}>
                {part.items.map((item, at) => (
                    <li key={at}>
                        <Runs runs={item.Runs} />
                    </li>
                ))}
            </ul>
        );
    }
    if (part.one.Form === 'quote') {
        return (
            <blockquote className={styles.quote}>
                <Runs runs={part.one.Runs} />
            </blockquote>
        );
    }
    return (
        <p className={styles.paragraph}>
            <Runs runs={part.one.Runs} />
        </p>
    );
}

export function Prose({ paragraphs }: { paragraphs: readonly WikiParagraph[] }) {
    return (
        <div className={styles.prose}>
            {partsOf(paragraphs).map((part, at) => (
                <Drawn key={at} part={part} />
            ))}
        </div>
    );
}
