import type { MouseEvent, ReactNode } from 'react';

import type { Tone } from '../tone';
import { StatePill } from './state-pill';
import { TypeTag } from './type-tag';
import { Value } from './value';

import styles from './row.module.css';

export type RowTemplate = 'catalog' | 'flow' | 'meters';

type RowState = { tone: Tone; word: string };

// value: a column given `null` is kept empty in this row, so a list whose rows differ still aligns; a column
// value: left `undefined` is not a column of this list at all. A value given as a string is plain text with
// value: tabular figures (a size, a count); a literal machine string comes as a `Value`, in mono, and so may
// value: the sentence when it is an address, cut in the middle.
export type RowProps = {
    title: string;
    tag?: string;
    sentence?: ReactNode;
    lead?: ReactNode;
    leadSize?: 'chip' | 'icon' | 'value';
    state?: null | RowState;
    stateAt?: 'end' | 'start';
    cells?: readonly ReactNode[];
    kind?: null | string;
    value?: ReactNode;
    valueSize?: 'long' | 'short';
    trail?: ReactNode;
    trailSize?: 'fit' | 'pill' | 'wide';
    size?: 'regular' | 'slim';
    template?: RowTemplate;
    onOpen?: () => void;
    link?: ReactNode;
};

type Columns = Pick<
    RowProps,
    | 'kind'
    | 'lead'
    | 'leadSize'
    | 'state'
    | 'stateAt'
    | 'template'
    | 'trail'
    | 'trailSize'
    | 'value'
    | 'valueSize'
>;

const LEAD_SIZES = { chip: styles.leadChip, icon: styles.leadIcon, value: styles.leadValue };
const TRAIL_SIZES = { fit: styles.hasTrail, pill: styles.trailPill, wide: styles.trailWide };

function leadOf({ lead, leadSize = 'icon' }: Columns): string | undefined {
    return lead === undefined ? undefined : LEAD_SIZES[leadSize];
}

function stateOf({ state, stateAt = 'end' }: Columns): string | undefined {
    if (state === undefined) {
        return undefined;
    }
    return stateAt === 'start' ? styles.hasStart : styles.hasState;
}

function valueOf({ value, valueSize = 'short' }: Columns): string | undefined {
    if (value === undefined) {
        return undefined;
    }
    return valueSize === 'long' ? styles.valueLong : styles.hasValue;
}

function columnsOf(columns: Columns): string[] {
    const { kind, trail, trailSize = 'fit', template = 'flow' } = columns;
    return [
        styles.grid,
        template === 'flow' ? undefined : styles[template],
        leadOf(columns),
        stateOf(columns),
        kind === undefined ? undefined : styles.hasKind,
        valueOf(columns),
        trail === undefined ? undefined : TRAIL_SIZES[trailSize],
    ].filter((name) => name !== undefined);
}

function classesOf(columns: Columns, size: 'regular' | 'slim', open: boolean): string {
    return [
        styles.row,
        size === 'slim' ? styles.slim : '',
        ...columnsOf(columns),
        open ? styles.open : '',
    ].join(' ');
}

function StateCell({ state, at }: { state: null | RowState | undefined; at: 'end' | 'start' }) {
    if (state === undefined) {
        return null;
    }
    return (
        <div className={`${styles.state} ${at === 'end' ? styles.atEnd : ''}`}>
            {state === null ? null : <StatePill quiet tone={state.tone} word={state.word} />}
        </div>
    );
}

function titledIfCut(event: MouseEvent<HTMLSpanElement>): void {
    const span = event.currentTarget;
    span.title = span.scrollWidth > span.clientWidth ? (span.textContent ?? '') : '';
}

type NameProps = { title: string; tag: string | undefined; sentence: ReactNode };

function Name({ title, tag, sentence }: NameProps) {
    const titled = (
        <span className={styles.title} onMouseEnter={titledIfCut}>
            {title}
        </span>
    );
    return (
        <div className={styles.text}>
            {tag === undefined ? (
                titled
            ) : (
                <span className={styles.tagged}>
                    {titled}
                    <Value look="raised">{tag}</Value>
                </span>
            )}
            {sentence === undefined || sentence === null || sentence === '' ? null : (
                <span className={styles.sentence} onMouseEnter={titledIfCut}>
                    {sentence}
                </span>
            )}
        </div>
    );
}

const NO_CELLS: readonly ReactNode[] = [];

function RowBody({
    title,
    tag,
    sentence,
    lead,
    state,
    stateAt = 'end',
    cells = NO_CELLS,
    kind,
    value,
    trail,
}: Omit<RowProps, 'link' | 'onOpen' | 'size' | 'trailSize'>) {
    return (
        <>
            {lead === undefined ? null : <div className={styles.lead}>{lead}</div>}
            {stateAt === 'start' ? <StateCell at="start" state={state} /> : null}
            <Name sentence={sentence} tag={tag} title={title} />
            {kind === undefined ? null : (
                <div className={styles.kind}>
                    {kind === null ? null : <TypeTag>{kind}</TypeTag>}
                </div>
            )}
            {cells.map((cell, index) => (
                <div className={styles.cell} key={index}>
                    {cell}
                </div>
            ))}
            {value === undefined ? null : (
                <div className={styles.value}>
                    {typeof value === 'string' ? (
                        <span className={styles.plain}>{value}</span>
                    ) : (
                        value
                    )}
                </div>
            )}
            {stateAt === 'end' ? <StateCell at="end" state={state} /> : null}
            {trail === undefined ? null : <div className={styles.end}>{trail}</div>}
        </>
    );
}

export function Row({ size = 'regular', onOpen, link, ...shown }: RowProps) {
    const body = <RowBody {...shown} />;
    const linked = link === undefined ? null : <div className={styles.link}>{link}</div>;
    if (onOpen && linked) {
        return (
            <div className={`${styles.row} ${size === 'slim' ? styles.slim : ''} ${styles.linked}`}>
                <button
                    className={`${styles.opener} ${columnsOf(shown).join(' ')}`}
                    onClick={onOpen}
                    type="button"
                >
                    {body}
                </button>
                {linked}
            </div>
        );
    }
    if (onOpen) {
        return (
            <button className={classesOf(shown, size, true)} onClick={onOpen} type="button">
                {body}
            </button>
        );
    }
    if (linked) {
        return (
            <div className={`${styles.row} ${size === 'slim' ? styles.slim : ''} ${styles.linked}`}>
                <div className={`${styles.inner} ${columnsOf(shown).join(' ')}`}>{body}</div>
                {linked}
            </div>
        );
    }
    return <div className={classesOf(shown, size, false)}>{body}</div>;
}

export type RowColumn = { label: string; numeric?: boolean };

export function RowHead({
    template,
    columns,
    lead = false,
}: {
    template: Exclude<RowTemplate, 'flow'>;
    columns: readonly RowColumn[];
    lead?: boolean;
}) {
    return (
        <div
            className={`${styles.row} ${styles.grid} ${styles[template]} ${lead ? styles.leadIcon : ''} ${styles.head}`}
        >
            {lead ? <span /> : null}
            {columns.map((column) => (
                <span
                    className={column.numeric === true ? styles.right : undefined}
                    key={column.label}
                >
                    {column.label}
                </span>
            ))}
        </div>
    );
}
