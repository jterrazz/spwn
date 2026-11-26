import { useId, useLayoutEffect, useRef, useState } from 'react';
import type { KeyboardEvent, MouseEvent, ReactNode } from 'react';

import type { Tone } from '../../domain/tone';
import { moved, moveOfKey } from './focus';
import { Icon } from './icon';
import { useMovedPointer } from './moved-pointer';
import { StateMark } from './state-mark';

import styles from './popover.module.css';

export type PopoverRow = {
    id: string;
    label: string;
    lead?: ReactNode;
    state?: undefined | { tone: Tone; word: string };
    description?: string | undefined;
    hint?: string | undefined;
    current?: boolean;
    indent?: boolean;
    disabled?: boolean;
};

// value: one group of rows under its caption; `meta` is said at the caption's end, `empty` in its place when a
// value: filter leaves it with no row, drawn only when the groups stand side by side.
export type PopoverSection = {
    label: string;
    rows: readonly PopoverRow[];
    meta?: string | undefined;
    empty?: string | undefined;
};

export type PopoverFilter = {
    label: string;
    placeholder: string;
    value: string;
    onChange: (value: string) => void;
};

export type PopoverListProps = {
    label: string;
    sections: readonly PopoverSection[];
    empty: string;
    filter?: PopoverFilter | undefined;
    columns?: boolean;
    onChoose: (id: string) => void;
    close: () => void;
};

function firstOf(rows: readonly PopoverRow[]): string | undefined {
    return (rows.find((row) => row.current === true) ?? rows.find((row) => row.disabled !== true))
        ?.id;
}

function startsOf(sections: readonly PopoverSection[], columns: boolean) {
    return sections
        .map((section, at) => ({
            section,
            start: sections.slice(0, at).reduce((sum, one) => sum + one.rows.length, 0),
        }))
        .filter(({ section }) => columns || section.rows.length > 0);
}

// invariant: Home and End move the caret while the filter holds the keyboard, as a text field's keys do; they
// invariant: move along the list only when the list itself holds it.
function moveOf(event: KeyboardEvent, filtered: boolean) {
    const caret = filtered && (event.key === 'Home' || event.key === 'End');
    return caret ? undefined : moveOfKey(event.key, 'vertical');
}

type OptionProps = {
    row: PopoverRow;
    id: string;
    active: boolean;
    onPick: () => void;
    onHover: (event: MouseEvent) => void;
};

function Option({ row, id, active, onPick, onHover }: OptionProps) {
    const said = row.description === undefined ? undefined : `${id}-said`;
    return (
        <div
            aria-describedby={said}
            aria-disabled={row.disabled === true ? true : undefined}
            aria-selected={row.current === true}
            className={`${styles.row} ${row.indent === true ? styles.indent : ''}`}
            data-active={active ? true : undefined}
            id={id}
            onClick={onPick}
            onMouseMove={onHover}
            role="option"
        >
            {row.lead}
            <span className={styles.words}>
                <span className={styles.label}>{row.label}</span>
                {said === undefined ? null : (
                    <span aria-hidden="true" className={styles.description} id={said}>
                        {row.description}
                    </span>
                )}
            </span>
            {row.hint === undefined ? null : <span className={styles.hint}>{row.hint}</span>}
            {row.state ? (
                <StateMark said="before" tone={row.state.tone} word={row.state.word} />
            ) : null}
        </div>
    );
}

function Filter({
    filter,
    list,
    active,
    onKeyDown,
}: {
    filter: PopoverFilter;
    list: string;
    active: string | undefined;
    onKeyDown: (event: KeyboardEvent) => void;
}) {
    return (
        <label className={styles.filter}>
            <Icon name="find" />
            <input
                aria-activedescendant={active}
                aria-autocomplete="list"
                aria-controls={list}
                aria-expanded="true"
                aria-label={filter.label}
                autoComplete="off"
                className={styles.input}
                onChange={(event) => {
                    filter.onChange(event.target.value);
                }}
                onKeyDown={onKeyDown}
                placeholder={filter.placeholder}
                role="combobox"
                spellCheck={false}
                type="text"
                value={filter.value}
            />
        </label>
    );
}

// value: the row the keyboard is on, kept by its id so a filter that keeps it keeps it, else the first row shown;
// value: a `typed` that changes, the columns' filter, starts it again on the first row it shows.
function useActiveRow(rows: readonly PopoverRow[], id: string, typed: string | undefined) {
    const [chosen, setChosen] = useState(() => firstOf(rows));
    const [was, setWas] = useState(typed);
    if (was !== typed) {
        setWas(typed);
        setChosen(rows.find((row) => row.disabled !== true)?.id);
    }
    const found = rows.findIndex((row) => row.id === chosen);
    const at = found !== -1 || rows.length === 0 ? found : 0;
    const active = at >= 0 ? `${id}-${String(at)}` : undefined;
    const list = useRef<HTMLDivElement>(null);
    useLayoutEffect(() => {
        if (active !== undefined) {
            list.current?.querySelector(`[id="${active}"]`)?.scrollIntoView({ block: 'nearest' });
        }
    }, [active]);
    const choose = (index: number) => {
        setChosen(rows[index]?.id);
    };
    return { active, at, choose, list };
}

type SectionsProps = {
    sections: readonly PopoverSection[];
    columns: boolean;
    id: string;
    at: number;
    onPick: (row: PopoverRow) => void;
    onHover: (event: MouseEvent, index: number) => void;
};

function Sections({ sections, columns, id, at, onPick, onHover }: SectionsProps) {
    return startsOf(sections, columns).map(({ section, start }) => (
        <div aria-label={section.label} className={styles.group} key={section.label} role="group">
            <p aria-hidden="true" className={styles.section}>
                {section.label}
                {section.meta === undefined ? null : <span>{section.meta}</span>}
            </p>
            {section.rows.length === 0 ? <p className={styles.none}>{section.empty}</p> : null}
            {section.rows.map((row, offset) => (
                <Option
                    active={start + offset === at}
                    id={`${id}-${String(start + offset)}`}
                    key={row.id}
                    onHover={(event) => {
                        onHover(event, start + offset);
                    }}
                    onPick={() => {
                        onPick(row);
                    }}
                    row={row}
                />
            ))}
        </div>
    ));
}

// invariant: with `columns` the sections stand side by side, each saying its own `empty` when a filter leaves it
// invariant: no row, and the arrows still walk one list, the first section top to bottom, then the next.
export function PopoverList(props: PopoverListProps) {
    const { label, sections, empty, filter, columns = false, onChoose, close } = props;
    const id = useId();
    const rows = sections.flatMap((section) => section.rows);
    const { active, at, choose, list } = useActiveRow(
        rows,
        id,
        columns ? filter?.value : undefined,
    );
    const hover = useMovedPointer(choose);
    const pick = (row: PopoverRow | undefined) => {
        if (row !== undefined && row.disabled !== true) {
            onChoose(row.id);
            close();
        }
    };
    const onKeyDown = (event: KeyboardEvent) => {
        const move = moveOf(event, filter !== undefined);
        if (move !== undefined) {
            event.preventDefault();
            choose(moved(rows.length, at, move, true));
        } else if (event.key === 'Enter') {
            event.preventDefault();
            pick(rows[at]);
        }
    };
    return (
        <>
            {filter ? (
                <Filter active={active} filter={filter} list={id} onKeyDown={onKeyDown} />
            ) : null}
            <div
                aria-activedescendant={filter ? undefined : active}
                aria-label={label}
                className={`${styles.list} ${columns ? styles.columns : ''}`}
                id={id}
                onKeyDown={onKeyDown}
                onMouseDown={(event) => {
                    event.preventDefault();
                }}
                ref={list}
                role="listbox"
                tabIndex={filter ? -1 : 0}
            >
                {rows.length === 0 ? (
                    <p className={styles.none}>{empty}</p>
                ) : (
                    <Sections
                        at={at}
                        columns={columns}
                        id={id}
                        onHover={hover}
                        onPick={pick}
                        sections={sections}
                    />
                )}
            </div>
        </>
    );
}
