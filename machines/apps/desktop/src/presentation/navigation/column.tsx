import type { EngineReadOutput, MachinesMacView, OverviewNeed } from '@jterrazz/station-api';
import { useEffect, useSyncExternalStore } from 'react';

import { needsYou } from '../../domain/overview/upkeep';
import type { Reading } from '../dependencies/dependencies';
import { Icon } from '../ui/icon';
import { IconButton, NavItem, StateDot } from '../ui/index';
import { wordOf } from '../words/engine-words';
import { needState } from '../words/shown';
import type { Carried } from './carried';
import { Machines, QuietMachines } from './column-machines';
import type { ColumnState } from './column-state';
import { EngineFoot } from './engine-foot';
import { onListKey } from './keys';
import { ReleaseRow } from './release-row';
import type { PlainPage, Route } from './route';

import styles from './column.module.css';

type Go = (to: Route) => void;

export type ColumnProps = {
    carried: Carried;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    engine: Reading<EngineReadOutput>;
    route: Route;
    go: Go;
    routeOfNeed: (need: OverviewNeed) => Route;
    quiet: boolean;
};

// value: how many needs the column lists before it sends the rest to the Inbox.
const NEEDS_LISTED = 3;

export const COLUMN_KEYS = 'Control+Meta+S';

const PIN_LABEL = 'Pin the column';

type Place = (page: PlainPage) => { current: boolean; disabled: boolean; onSelect: () => void };

function placeOf(route: Route, go: Go, quiet: boolean): Place {
    return (page) => ({
        current: !quiet && route.page === page,
        disabled: quiet,
        onSelect: () => {
            go({ page });
        },
    });
}

function useColumn(column: ColumnState) {
    return useSyncExternalStore(column.subscribe, column.read);
}

export function ColumnToggle({ column }: { column: ColumnState }) {
    const { pinned } = useColumn(column);
    return (
        <IconButton
            icon="sidebar"
            label={PIN_LABEL}
            onClick={column.toggle}
            pressed={pinned}
            shortcut="⌃⌘S"
        />
    );
}

function Needs({ needs, go, routeOfNeed }: Pick<ColumnProps, 'go' | 'needs' | 'routeOfNeed'>) {
    const owed = needsYou(needs);
    if (owed.length === 0) {
        return null;
    }
    const more = owed.length - NEEDS_LISTED;
    return (
        <div aria-labelledby="column-needs" className={styles.group} role="group">
            <div className={styles.caption} id="column-needs">
                <span>{wordOf('OverviewUpkeep', 'needs-you').Word}</span>
                <span className={styles.owed}>{owed.length}</span>
            </div>
            {owed.slice(0, NEEDS_LISTED).map((need) => (
                <NavItem
                    current={false}
                    key={`${need.Machine}/${need.Section}/${need.Subject}`}
                    label={need.Text}
                    lead={
                        <span aria-hidden="true" className={styles.need}>
                            <StateDot {...needState(need)} />
                        </span>
                    }
                    onSelect={() => {
                        go(routeOfNeed(need));
                    }}
                />
            ))}
            {more > 0 ? (
                <NavItem
                    current={false}
                    icon="inbox"
                    label={`${String(more)} more`}
                    onSelect={() => {
                        go({ page: 'inbox' });
                    }}
                />
            ) : null}
        </div>
    );
}

function countOf(value: number | undefined) {
    return value === undefined ? undefined : { urgent: false, value };
}

function Library({ place, carried }: { place: Place; carried: Carried }) {
    return (
        <div aria-labelledby="column-library" className={styles.group} role="group">
            <div className={styles.caption} id="column-library">
                Library
            </div>
            <NavItem
                {...place('catalog-apps')}
                count={countOf(carried.apps)}
                icon="apps"
                label="Apps"
            />
            <NavItem
                {...place('catalog-skills')}
                count={countOf(carried.skills)}
                icon="skills"
                label="Skills"
            />
            <NavItem
                {...place('catalog-settings')}
                count={countOf(carried.settings)}
                icon="settings"
                label="Machine settings"
            />
            <NavItem {...place('machine-files')} icon="file" label="Machine files" />
            <NavItem {...place('storage')} icon="disc" label="Storage" />
            <NavItem {...place('passwords')} icon="key" label="Passwords" />
        </div>
    );
}

const UNCARRIED: Carried = { apps: undefined, settings: undefined, skills: undefined };

// value: the Machine space's column: what needs me, the machines under their Macs, then the Library; the engine's
// value: line and a release at its foot.
function Column(props: ColumnProps & { pin?: (() => void) | undefined }) {
    const { carried, macs, needs, engine, route, go, routeOfNeed, quiet, pin } = props;
    const place = placeOf(route, go, quiet);
    return (
        <nav aria-label="Column" className={styles.column} onKeyDown={onListKey}>
            <div className={styles.list}>
                {quiet ? null : <Needs go={go} needs={needs} routeOfNeed={routeOfNeed} />}
                {quiet ? <QuietMachines /> : <Machines go={go} macs={macs} route={route} />}
                <Library carried={quiet ? UNCARRIED : carried} place={place} />
            </div>
            <EngineFoot engine={engine} go={go} quiet={quiet} />
            {quiet ? null : <ReleaseRow go={go} />}
            {pin === undefined ? null : (
                <button
                    aria-keyshortcuts={COLUMN_KEYS}
                    className={styles.pin}
                    onClick={pin}
                    type="button"
                >
                    <Icon name="pin" />
                    <span className={styles.pinLabel}>{PIN_LABEL}</span>
                    <kbd aria-hidden="true">⌃⌘S</kbd>
                </button>
            )}
        </nav>
    );
}

// invariant: pinned, the column takes its place beside the content; hidden, an edge at the window's left peeks it
// invariant: over the content, opaque, until the pointer has left it; a choice made in a peek closes it.
export function ColumnSlot({ column, ...props }: ColumnProps & { column: ColumnState }) {
    const { pinned, peeking } = useColumn(column);
    useEffect(() => {
        const onResize = () => {
            column.resized(globalThis.innerWidth);
        };
        globalThis.addEventListener('resize', onResize);
        return () => {
            globalThis.removeEventListener('resize', onResize);
        };
    }, [column]);
    if (pinned) {
        return (
            <div className={styles.pinned}>
                <Column {...props} />
            </div>
        );
    }
    const go: Go = (to) => {
        column.close();
        props.go(to);
    };
    return (
        <div className={styles.hidden}>
            <div
                aria-hidden="true"
                className={styles.edge}
                data-testid="column-edge"
                onPointerEnter={column.peek}
                onPointerLeave={column.leave}
            />
            {peeking ? (
                <div
                    className={styles.peek}
                    onPointerEnter={column.stay}
                    onPointerLeave={column.leave}
                >
                    <Column {...props} go={go} pin={column.pin} />
                </div>
            ) : null}
        </div>
    );
}
