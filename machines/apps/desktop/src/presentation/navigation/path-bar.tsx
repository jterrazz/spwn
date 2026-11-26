import type { EngineReadOutput, MachinesMacView, OverviewNeed } from '@jterrazz/station-api';
import type { ComponentType } from 'react';

import type { Found } from '../../domain/machines/machines';
import { needsYou } from '../../domain/overview/upkeep';
import type { Reading } from '../dependencies/dependencies';
import { Icon } from '../ui/icon';
import { Button, IconButton, Sign } from '../ui/index';
import type { Shown } from '../words/shown';
import { actionOf } from './bar-action';
import type { Card } from './card';
import type { Carried } from './carried';
import { ColumnToggle } from './column';
import type { ColumnState } from './column-state';
import { EngineMark } from './engine-foot';
import { PathNav } from './path';
import type { PlainPage } from './route';
import type { Router } from './router';
import { segmentsOf } from './segments';
import type { Soon } from './soon';
import { SpaceSwitch } from './space-switch';
import type { Space } from './spaces';
import { StaleMark } from './stale-mark';

import styles from './path-bar.module.css';

export type PathBarProps = {
    router: Router;
    quiet: boolean;
    found: Found | undefined;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    engine: Reading<EngineReadOutput>;
    standing: Shown | undefined;
    onSpace: (space: Space) => void;
    soon: Soon;
    card: Card;
    carried: Carried;
    onFind: () => void;
    column: ColumnState | undefined;
    onAddPassword: () => void;
    onNewMachine: () => void;
    ItemName: ComponentType<{ page: PlainPage; item: string }>;
};

const NOWHERE = () => undefined;

function Inbox({
    needs,
    quiet,
    go,
}: {
    needs: readonly OverviewNeed[];
    quiet: boolean;
    go: () => void;
}) {
    const owed = quiet ? [] : needsYou(needs);
    const urgent = owed.some((need) => need.Broken);
    return (
        <button
            aria-keyshortcuts="Meta+I"
            aria-label={owed.length > 0 ? `Inbox ${String(owed.length)}` : 'Inbox'}
            className={styles.inbox}
            disabled={quiet}
            onClick={go}
            title="Inbox (⌘I)"
            type="button"
        >
            <Icon name="inbox" />
            {owed.length > 0 ? (
                <span
                    aria-hidden="true"
                    className={`${styles.count} ${urgent ? styles.urgent : ''}`}
                >
                    {owed.length}
                </span>
            ) : null}
        </button>
    );
}

function End({ router, quiet, macs, needs, engine, onFind, ...props }: PathBarProps) {
    const { route, go } = router;
    const action = quiet
        ? undefined
        : actionOf({
              addPassword: props.onAddPassword,
              macs,
              needs,
              newMachine: props.onNewMachine,
              route,
          });
    return (
        <div className={styles.end}>
            <EngineMark engine={engine} go={go} quiet={quiet} />
            {quiet ? null : <StaleMark />}
            {action === undefined ? null : (
                <Button onClick={action.onClick} variant={action.leads ? 'primary' : 'plain'}>
                    {action.label}
                </Button>
            )}
            <Inbox
                go={() => {
                    go({ page: 'inbox' });
                }}
                needs={needs}
                quiet={quiet}
            />
            <button
                aria-keyshortcuts="Meta+K"
                aria-label="Find"
                className={styles.find}
                disabled={quiet}
                onClick={onFind}
                type="button"
            >
                <Icon name="find" />
                <span aria-hidden="true">Find</span>
                <kbd aria-hidden="true">⌘K</kbd>
            </button>
            <IconButton
                disabled={quiet}
                icon="gear"
                label="Settings"
                onClick={() => {
                    go({ page: 'settings' });
                }}
                shortcut="⌘,"
            />
        </div>
    );
}

// invariant: the bar is the window's one row of navigation: the spaces and their rule, the sign, the column's pin,
// invariant: Back and Forward, the path, then the engine's mark, the stale mark, the page's action, Inbox, Find and
// invariant: Settings. It is a deep drag region, which leaves every button in it clickable; its popovers draw in a
// invariant: portal outside it.
export function PathBar(props: PathBarProps) {
    const { router, quiet, found, macs, standing, onSpace, soon, column, ItemName } = props;
    const { card, carried, engine } = props;
    const { route, go, back, forward, canGoBack, canGoForward } = router;
    const segments = segmentsOf({ found, route }, quiet);
    return (
        <header className={styles.bar} data-tauri-drag-region="deep">
            <SpaceSwitch
                onSpace={onSpace}
                quiet={quiet}
                route={route}
                soon={soon}
                standing={standing}
            />
            <button
                aria-keyshortcuts="Meta+0"
                aria-label="Home"
                className={styles.sign}
                onClick={() => {
                    go({ page: 'home' });
                }}
                title="Home (⌘0)"
                type="button"
            >
                <Sign />
            </button>
            {column === undefined ? null : <ColumnToggle column={column} />}
            <IconButton
                disabled={quiet || !canGoBack}
                icon="back"
                label="Back"
                onClick={quiet ? NOWHERE : back}
                shortcut="⌘["
            />
            <IconButton
                disabled={quiet || !canGoForward}
                icon="forward"
                label="Forward"
                onClick={quiet ? NOWHERE : forward}
                shortcut="⌘]"
            />
            <PathNav
                card={card}
                carried={carried}
                engine={engine}
                found={found}
                go={go}
                ItemName={ItemName}
                macs={macs}
                route={route}
                segments={segments}
            />
            <End {...props} />
        </header>
    );
}
