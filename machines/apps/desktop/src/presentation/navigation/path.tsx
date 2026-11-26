import type { EngineReadOutput, MachinesMacView } from '@jterrazz/station-api';
import { Fragment, useId, useLayoutEffect, useRef, useSyncExternalStore } from 'react';
import type { ComponentType, KeyboardEvent, ReactNode, RefObject } from 'react';

import { machineNamesOf } from '../../domain/machines/machines';
import type { Found } from '../../domain/machines/machines';
import type { Reading } from '../dependencies/dependencies';
import { focusablesIn, moveFocus, moveOfKey } from '../ui/focus';
import { Icon } from '../ui/icon';
import { identityOf, MachineIcon, StateMark, usePopover } from '../ui/index';
import { machineKindOf } from '../words/provisional';
import type { Card } from './card';
import type { Carried } from './carried';
import { MachinePopover } from './machine-popover';
import type { Opened } from './machine-popover';
import { WaysPopover } from './place-popover';
import type { Way } from './place-popover';
import { machineRoute } from './route';
import type { PlainPage, Route } from './route';
import type { Segment } from './segments';
import { SpaceCard } from './space-card';
import { keyOf, useFold } from './use-fold';

import styles from './path-bar.module.css';

export type PathProps = {
    segments: readonly Segment[];
    found: Found | undefined;
    macs: readonly MachinesMacView[];
    route: Route;
    go: (to: Route) => void;
    card: Card;
    carried: Carried;
    engine: Reading<EngineReadOutput>;
    ItemName: ComponentType<{ page: PlainPage; item: string }>;
};

type Opens = Omit<PathProps, 'ItemName' | 'segments'>;

function leadOf(segment: Segment, { found, macs }: Pick<Opens, 'found' | 'macs'>): ReactNode {
    if (segment.key === 'vm' && found?.vm !== undefined) {
        return (
            <MachineIcon
                identity={identityOf(found.vm.Name, machineNamesOf(macs))}
                kind={machineKindOf(found.vm)}
            />
        );
    }
    return segment.key === 'mac' || segment.key === 'machine' ? <MachineIcon kind="mac" /> : null;
}

function currentOf(route: Route): string | undefined {
    return route.page === 'machine' ? route.machine : route.page;
}

type SiblingsProps = { segment: Segment; opened: Opened; opens: Opens };

function SiblingsOf({ segment, opened, opens }: SiblingsProps) {
    const { macs, route, go } = opens;
    const current = currentOf(route);
    const toMachine = (machine: string) => {
        go(machineRoute(machine));
    };
    switch (segment.siblings) {
        case 'fleet': {
            return (
                <MachinePopover {...opened} current={current} macs={macs} onChoose={toMachine} />
            );
        }
        case 'vms': {
            return (
                <MachinePopover
                    {...opened}
                    current={current}
                    mac={segment.mac}
                    macs={macs}
                    onChoose={toMachine}
                />
            );
        }
        case 'card':
        case undefined: {
            return null;
        }
    }
}

type Stop = { stop: boolean; last: boolean };

function Face({ segment, opens }: { segment: Segment; opens: Opens }) {
    return (
        <>
            {leadOf(segment, opens)}
            <span className={styles.name} data-fits="">
                {segment.label}
            </span>
            {segment.shown === undefined ? null : (
                <StateMark said="name" tone={segment.shown.tone} word={segment.shown.word} />
            )}
            <span aria-hidden="true" className={styles.chevron}>
                <Icon name="choose" />
            </span>
        </>
    );
}

function Opener({ segment, opens, stop, last }: Stop & { segment: Segment; opens: Opens }) {
    const { anchor, close, id, open, opener } = usePopover();
    return (
        <>
            <button
                {...opener}
                aria-current={last ? 'page' : undefined}
                className={styles.segment}
                tabIndex={stop ? 0 : -1}
                type="button"
            >
                <Face opens={opens} segment={segment} />
            </button>
            {open ? (
                <SiblingsOf
                    opened={{ anchor, id, onClose: close }}
                    opens={opens}
                    segment={segment}
                />
            ) : null}
        </>
    );
}

// invariant: the card opens from its segment whatever opened it, ⌘L included: the segment holds the focus before
// invariant: the card takes it, so Escape returns it there.
function FromSegment({ anchor }: { anchor: RefObject<HTMLButtonElement | null> }) {
    useLayoutEffect(() => {
        anchor.current?.focus({ preventScroll: true });
    }, [anchor]);
    return null;
}

// value: the first segment, the section, which opens the card by a press, ArrowDown or ⌘L; whether the card is
// value: open is the window's Card, so opening it draws this segment and the card alone.
function CardOpener({ segment, opens, stop, last }: Stop & { segment: Segment; opens: Opens }) {
    const { card } = opens;
    const open = useSyncExternalStore(card.subscribe, card.shown);
    const anchor = useRef<HTMLButtonElement>(null);
    const id = useId();
    return (
        <>
            <button
                aria-controls={open ? id : undefined}
                aria-current={last ? 'page' : undefined}
                aria-expanded={open}
                aria-haspopup="dialog"
                aria-keyshortcuts="Meta+L"
                className={styles.segment}
                onClick={card.toggle}
                onKeyDown={(event) => {
                    if (event.key === 'ArrowDown') {
                        event.preventDefault();
                        card.show();
                    }
                }}
                ref={anchor}
                tabIndex={stop ? 0 : -1}
                title={`${segment.label} (⌘L)`}
                type="button"
            >
                <Face opens={opens} segment={segment} />
            </button>
            {open ? <FromSegment anchor={anchor} /> : null}
            {open ? (
                <SpaceCard
                    anchor={anchor}
                    carried={opens.carried}
                    engine={opens.engine}
                    go={opens.go}
                    id={id}
                    label={segment.label}
                    lastMachine={card.lastMachine}
                    macs={opens.macs}
                    onClose={card.close}
                    route={opens.route}
                />
            ) : null}
        </>
    );
}

function Fold({
    hidden,
    opens,
    stop,
}: {
    hidden: readonly Segment[];
    opens: Opens;
    stop: boolean;
}) {
    const { anchor, close, id, open, opener } = usePopover();
    const ways: Way[] = hidden.flatMap((segment) =>
        segment.to === undefined ? [] : [{ id: segment.key, label: segment.label, to: segment.to }],
    );
    return (
        <>
            <button
                {...opener}
                aria-label="Folded levels"
                className={styles.segment}
                tabIndex={stop ? 0 : -1}
                type="button"
            >
                …
            </button>
            {open ? (
                <WaysPopover
                    anchor={anchor}
                    go={opens.go}
                    id={id}
                    label="Folded levels"
                    onClose={close}
                    ways={ways}
                />
            ) : null}
        </>
    );
}

function onPathKey(event: KeyboardEvent<HTMLElement>): void {
    const move = moveOfKey(event.key, 'horizontal');
    if (move === 'next' || move === 'previous') {
        event.preventDefault();
        moveFocus(focusablesIn(event.currentTarget, 'button'), move, false);
    }
}

type LevelProps = Stop & { segment: Segment; opens: Opens; ItemName: PathProps['ItemName'] };

function Level({ segment, opens, stop, last, ItemName }: LevelProps) {
    if (segment.siblings === 'card') {
        return <CardOpener last={last} opens={opens} segment={segment} stop={stop} />;
    }
    if (segment.siblings !== undefined) {
        return <Opener last={last} opens={opens} segment={segment} stop={stop} />;
    }
    return (
        <span aria-current={last ? 'page' : undefined} className={styles.words} data-fits="">
            {segment.item === undefined ? (
                segment.label
            ) : (
                <ItemName item={segment.item.item} page={segment.item.page} />
            )}
        </span>
    );
}

function Slash() {
    return (
        <span aria-hidden="true" className={styles.slash}>
            /
        </span>
    );
}

// value: the path: one segment per level, each a press away from its siblings; the levels between the first and
// value: the last fold into one `…` when the bar has no room for them.
export function PathNav({ segments, ItemName, ...opens }: PathProps) {
    const { folded, nav } = useFold(segments);
    const hidden = segments.filter((segment) => folded.includes(keyOf(segment)));
    const shown = segments.filter((segment) => !folded.includes(keyOf(segment)));
    const stopAt = shown.findLastIndex((segment) => segment.siblings !== undefined);
    return (
        <nav aria-label="Path" className={styles.path} ref={nav}>
            <ol className={styles.levels} onKeyDown={onPathKey}>
                {shown.map((segment, at) => (
                    <Fragment key={keyOf(segment)}>
                        {at === 1 && hidden.length > 0 ? (
                            <li className={styles.level}>
                                <Slash />
                                <Fold hidden={hidden} opens={opens} stop={stopAt === -1} />
                            </li>
                        ) : null}
                        <li className={styles.level} data-segment={keyOf(segment)}>
                            {at === 0 ? null : <Slash />}
                            <Level
                                ItemName={ItemName}
                                last={at === shown.length - 1}
                                opens={opens}
                                segment={segment}
                                stop={at === stopAt}
                            />
                        </li>
                    </Fragment>
                ))}
            </ol>
        </nav>
    );
}
