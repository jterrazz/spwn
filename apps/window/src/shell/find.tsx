import type { MachinesMacView, OverviewNeed } from '@jterrazz/station-api';
import { useId, useLayoutEffect, useRef, useState } from 'react';
import type { KeyboardEvent, MouseEvent } from 'react';

import { moved } from '../design/focus';
import { AppIcon, identityOf, MachineIcon, Modal } from '../design/index';
import { isSkill } from '../engine/provisional';
import { useWatch } from '../engine/react';
import { machineNamesOf } from '../model/machines';
import { everything, matching, resultKey } from './finds';
import type { Group, Result } from './finds';
import type { PlainPage, Route } from './route';

import styles from './find.module.css';

export type FindProps = {
    titles: Readonly<Record<PlainPage, string>>;
    withGallery: boolean;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    routeOfNeed: (need: OverviewNeed) => Route;
    go: (to: Route) => void;
    onClose: () => void;
};

function useHeld(props: FindProps) {
    const apps = useWatch('apps.list', {});
    const settings = useWatch('settings.list', {});
    return everything({
        ...props,
        apps:
            apps.kind === 'read'
                ? (apps.out ?? []).map(({ App }) => ({
                      category: App.Category,
                      name: App.Name,
                      skill: isSkill(App.Category),
                  }))
                : [],
        settings:
            settings.kind === 'read'
                ? (settings.out ?? []).map((row) => ({
                      category: row.Package.Category,
                      name: row.Name,
                  }))
                : [],
    });
}

const KEYS: Readonly<Record<string, 'first' | 'last' | 'next' | 'previous'>> = {
    ArrowDown: 'next',
    ArrowUp: 'previous',
    PageDown: 'last',
    PageUp: 'first',
};

type ListProps = {
    groups: readonly Group[];
    chosen: number;
    id: string;
    listed: readonly string[];
    onPick: (result: Result) => void;
    onHover: (event: MouseEvent, index: number) => void;
};

type Pointer = { x: number; y: number };

// hazard: Chromium sends a mousemove to whatever comes to lie under a resting pointer when the list redraws, so
// hazard: only a pointer that moved since its last event, and not the first event heard, chooses a row.
function useHover(onChoose: (index: number) => void): (event: MouseEvent, index: number) => void {
    const last = useRef<Pointer | undefined>(undefined);
    return (event, index) => {
        const before = last.current;
        last.current = { x: event.clientX, y: event.clientY };
        if (before !== undefined && (before.x !== event.clientX || before.y !== event.clientY)) {
            onChoose(index);
        }
    };
}

function startsOf(groups: readonly Group[]): readonly number[] {
    return groups.map((_, at) =>
        groups.slice(0, at).reduce((sum, group) => sum + group.results.length, 0),
    );
}

function List({ groups, chosen, id, listed, onPick, onHover }: ListProps) {
    const starts = startsOf(groups);
    return (
        <div aria-label="Results" className={styles.list} id={id} role="listbox">
            {groups.map((group, place) => (
                <div aria-label={group.kind} key={group.kind} role="group">
                    {group.results.map((result, offset) => {
                        const at = (starts[place] ?? 0) + offset;
                        return (
                            <div
                                aria-selected={at === chosen}
                                className={styles.result}
                                id={`${id}-${String(at)}`}
                                key={resultKey(result)}
                                onClick={() => {
                                    onPick(result);
                                }}
                                onMouseMove={(event) => {
                                    onHover(event, at);
                                }}
                                role="option"
                            >
                                {result.mark === undefined ? null : (
                                    <AppIcon kind={result.mark.kind} name={result.mark.name} />
                                )}
                                {result.machine === undefined ? null : (
                                    <MachineIcon
                                        identity={identityOf(result.machine.name, listed)}
                                        kind={result.machine.kind}
                                    />
                                )}
                                <span className={styles.label}>{result.label}</span>
                                <span className={styles.detail}>{result.detail}</span>
                            </div>
                        );
                    })}
                </div>
            ))}
            {groups.length === 0 ? <p className={styles.none}>Nothing matches</p> : null}
        </div>
    );
}

type QueryProps = {
    value: string;
    list: string;
    active: string | undefined;
    onChange: (value: string) => void;
    onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
};

function Query({ value, list, active, onChange, onKeyDown }: QueryProps) {
    return (
        <input
            aria-activedescendant={active}
            aria-autocomplete="list"
            aria-controls={list}
            aria-expanded="true"
            aria-label="Find"
            autoComplete="off"
            className={styles.input}
            onChange={(event) => {
                onChange(event.target.value);
            }}
            onKeyDown={onKeyDown}
            placeholder="Find a machine or a page"
            role="combobox"
            spellCheck={false}
            type="text"
            value={value}
        />
    );
}

export function Find(props: FindProps) {
    const { go, onClose } = props;
    const all = useHeld(props);
    const [query, setQuery] = useState('');
    const [chosen, setChosen] = useState(0);
    const id = useId();
    const list = useRef<HTMLDivElement>(null);
    const hover = useHover(setChosen);
    const groups = matching(all, query);
    const flat = groups.flatMap((group) => group.results);
    const at = Math.min(chosen, flat.length - 1);
    useLayoutEffect(() => {
        list.current
            ?.querySelector(`[id="${id}-${String(at)}"]`)
            ?.scrollIntoView({ block: 'nearest' });
    }, [at, id]);
    const pick = (result: Result) => {
        onClose();
        go(result.to.route);
    };
    const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
        const move = KEYS[event.key];
        if (move !== undefined) {
            event.preventDefault();
            setChosen(moved(flat.length, at, move, true));
            return;
        }
        const result = flat[at];
        if (event.key === 'Enter' && result) {
            event.preventDefault();
            pick(result);
        }
    };
    return (
        <Modal className={styles.pop} label="Find" onClose={onClose}>
            <Query
                active={at >= 0 ? `${id}-${String(at)}` : undefined}
                list={id}
                onChange={(value) => {
                    setQuery(value);
                    setChosen(0);
                }}
                onKeyDown={onKeyDown}
                value={query}
            />
            <div ref={list}>
                <List
                    chosen={at}
                    groups={groups}
                    id={id}
                    listed={machineNamesOf(props.macs)}
                    onHover={hover}
                    onPick={pick}
                />
            </div>
        </Modal>
    );
}
