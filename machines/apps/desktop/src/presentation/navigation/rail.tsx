import type { EngineReadOutput, OverviewNeed } from '@jterrazz/station-api';
import { useId } from 'react';

import { needsYou } from '../../domain/overview/upkeep';
import { useCore, useReach } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { Icon } from '../ui/icon';
import type { IconName } from '../ui/icon';
import { StateDot } from '../ui/index';
import { fitFootWord, fitState } from '../words/fit';
import { coreState, ownState } from '../words/shown';
import type { Shown } from '../words/shown';
import { CORE_FOOT_WORDS } from '../words/words';
import { useEngineLine } from './engine-line';
import type { Route } from './route';
import { inSpace, opensNow, SOON_WORD, SPACE_WORDS, SPACES } from './spaces';
import type { Space } from './spaces';

import styles from './rail.module.css';

export type RailProps = {
    route: Route;
    go: (to: Route) => void;
    onSpace: (space: Space) => void;
    needs: readonly OverviewNeed[];
    engine: Reading<EngineReadOutput>;
    quiet: boolean;
};

const SPACE_ICONS: Readonly<Record<Space, IconName>> = {
    agent: 'agent',
    machine: 'machine',
    os: 'os',
};

const SPACE_KEYS: Readonly<Record<Space, { aria: string; shown: string }>> = {
    agent: { aria: 'Control+2', shown: '⌃2' },
    machine: { aria: 'Control+1', shown: '⌃1' },
    os: { aria: 'Control+3', shown: '⌃3' },
};

type Keys = { aria: string; shown: string };

type TipProps = { id: string; label: string; keys?: Keys | undefined; read?: boolean };

function Tip({ id, label, keys, read = false }: TipProps) {
    return (
        <span aria-hidden={read ? undefined : true} className={styles.tip}>
            <span className={styles.tipHead}>
                <span id={id}>{label}</span>
                {keys === undefined ? null : (
                    <kbd aria-hidden="true" className={styles.tipKeys}>
                        {keys.shown}
                    </kbd>
                )}
            </span>
        </span>
    );
}

type PlaceProps = {
    icon: IconName;
    label: string;
    current: boolean;
    disabled: boolean;
    keys?: Keys | undefined;
    onSelect: () => void;
    count?: undefined | { value: number; urgent: boolean };
};

function Place({ icon, label, current, disabled, keys, onSelect, count }: PlaceProps) {
    const id = useId();
    const named = count === undefined ? `${id}-name` : `${id}-name ${id}-count`;
    return (
        <span className={styles.slot}>
            <button
                aria-current={current ? 'page' : undefined}
                aria-keyshortcuts={keys?.aria}
                aria-labelledby={named}
                className={styles.place}
                disabled={disabled}
                onClick={onSelect}
                type="button"
            >
                <Icon name={icon} />
                {count === undefined ? null : (
                    <span
                        aria-hidden="true"
                        className={`${styles.count} ${count.urgent ? styles.urgent : ''}`}
                        id={`${id}-count`}
                    >
                        {count.value}
                    </span>
                )}
            </button>
            <Tip id={`${id}-name`} keys={keys} label={label} />
        </span>
    );
}

function Segment({
    space,
    chosen,
    active,
    onSpace,
}: {
    space: Space;
    chosen: boolean;
    active: boolean;
    onSpace: RailProps['onSpace'];
}) {
    const words = SPACE_WORDS[space];
    const soon = !opensNow(space);
    const tip = `rail-tip-${space}`;
    return (
        <span className={styles.slot}>
            <button
                aria-describedby={soon ? `${tip}-line` : undefined}
                aria-disabled={soon ? true : undefined}
                aria-keyshortcuts={SPACE_KEYS[space].aria}
                aria-labelledby={soon ? `${tip}-name ${tip}-soon` : `${tip}-name`}
                aria-pressed={chosen}
                className={`${styles.place} ${soon ? styles.soon : ''} ${chosen && active ? styles.chosen : ''}`}
                onClick={() => {
                    if (!soon) {
                        onSpace(space);
                    }
                }}
                type="button"
            >
                <Icon name={SPACE_ICONS[space]} />
            </button>
            {soon ? (
                <span aria-hidden="true" className={styles.tip} role="tooltip">
                    <span className={styles.tipHead}>
                        <span id={`${tip}-name`}>{words.name}</span>
                        <span className={styles.pill} id={`${tip}-soon`}>
                            {SOON_WORD.toLowerCase()}
                        </span>
                        <kbd aria-hidden="true" className={styles.tipKeys}>
                            {SPACE_KEYS[space].shown}
                        </kbd>
                    </span>
                    <span id={`${tip}-line`}>{words.line}</span>
                </span>
            ) : (
                <Tip id={`${tip}-name`} keys={SPACE_KEYS[space]} label={words.name} />
            )}
        </span>
    );
}

function Spaces({ route, onSpace, quiet }: Pick<RailProps, 'onSpace' | 'quiet' | 'route'>) {
    const active = !quiet && inSpace(route);
    return (
        <div aria-label="Spaces" className={styles.places} role="group">
            {SPACES.map((space) => (
                <Segment
                    active={active}
                    chosen={space === 'machine'}
                    key={space}
                    onSpace={onSpace}
                    space={space}
                />
            ))}
        </div>
    );
}

function Foot({
    dot,
    said,
    below,
    go,
}: {
    dot: Shown;
    said: string;
    below: string;
    go?: (() => void) | undefined;
}) {
    const id = useId();
    const inner = (
        <>
            <span aria-hidden={go === undefined ? undefined : true} className={styles.dot}>
                <StateDot tone={dot.tone} word={dot.word} />
            </span>
            <span aria-hidden="true" className={styles.version}>
                {below}
            </span>
        </>
    );
    return (
        <span className={styles.slot}>
            {go === undefined ? (
                <div className={styles.engine}>{inner}</div>
            ) : (
                <button aria-labelledby={id} className={styles.engine} onClick={go} type="button">
                    {inner}
                </button>
            )}
            <Tip id={id} label={said} read={go === undefined} />
        </span>
    );
}

function QuietEngine() {
    const { state } = useCore();
    const reach = useReach();
    if (reach.kind === 'incompatible') {
        return <Foot below="" dot={fitState(reach.fit)} said={fitFootWord(reach.fit)} />;
    }
    const dot = state.kind === 'installed' ? ownState('not-started') : coreState(state.kind);
    return <Foot below="" dot={dot} said={CORE_FOOT_WORDS[state.kind]} />;
}

function Engine({ engine, go }: Pick<RailProps, 'engine' | 'go'>) {
    const line = useEngineLine(engine);
    return (
        <Foot
            below={line.version}
            dot={line.dot}
            go={
                line.opens
                    ? () => {
                          go({ page: 'engine' });
                      }
                    : undefined
            }
            said={line.said}
        />
    );
}

function owedOf(needs: readonly OverviewNeed[]): PlaceProps['count'] {
    const owed = needsYou(needs);
    return owed.length > 0
        ? { urgent: owed.some((need) => need.Broken), value: owed.length }
        : undefined;
}

type Reached = 'home' | 'inbox' | 'passwords' | 'settings';

// invariant: the rail is the window's, whatever the space: Home and Inbox at its top, the spaces one choice of three
// invariant: in its middle, the global pages and the engine at its foot.
export function Rail({ route, go, onSpace, needs, engine, quiet }: RailProps) {
    const at = (page: Reached) => ({
        current: quiet ? page === 'home' : route.page === page,
        disabled: quiet && page !== 'home',
        onSelect: () => {
            go({ page });
        },
    });
    return (
        <nav aria-label="Rail" className={styles.rail}>
            <div className={styles.lights} data-tauri-drag-region="deep" />
            <div className={styles.places}>
                <Place
                    {...at('home')}
                    icon="home"
                    keys={{ aria: 'Meta+0', shown: '⌘0' }}
                    label="Home"
                />
                <Place
                    {...at('inbox')}
                    count={quiet ? undefined : owedOf(needs)}
                    icon="inbox"
                    keys={{ aria: 'Meta+I', shown: '⌘I' }}
                    label="Inbox"
                />
            </div>
            <span aria-hidden="true" className={styles.rule} />
            <Spaces onSpace={onSpace} quiet={quiet} route={route} />
            <div className={styles.foot}>
                <Place {...at('passwords')} icon="key" label="Passwords" />
                <Place
                    {...at('settings')}
                    icon="gear"
                    keys={{ aria: 'Meta+,', shown: '⌘,' }}
                    label="Settings"
                />
                {quiet ? <QuietEngine /> : <Engine engine={engine} go={go} />}
            </div>
        </nav>
    );
}
