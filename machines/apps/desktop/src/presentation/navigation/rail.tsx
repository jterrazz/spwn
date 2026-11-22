import type { EngineReadOutput, OverviewNeed } from '@jterrazz/station-api';

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

const SPACE_KEYS: Readonly<Record<Space, string>> = { agent: '⌃2', machine: '⌃1', os: '⌃3' };

type PlaceProps = {
    icon: IconName;
    label: string;
    current: boolean;
    disabled: boolean;
    keys: string;
    onSelect: () => void;
    count?: undefined | { value: number; urgent: boolean };
};

function Place({ icon, label, current, disabled, keys, onSelect, count }: PlaceProps) {
    return (
        <button
            aria-current={current ? 'page' : undefined}
            aria-keyshortcuts={keys}
            className={styles.place}
            disabled={disabled}
            onClick={onSelect}
            type="button"
        >
            <span className={styles.mark}>
                <Icon name={icon} />
            </span>
            <span className={styles.label}>{label}</span>
            {count === undefined ? null : (
                <span className={`${styles.count} ${count.urgent ? styles.urgent : ''}`}>
                    {count.value}
                </span>
            )}
        </button>
    );
}

function Segment({
    space,
    chosen,
    onSpace,
}: {
    space: Space;
    chosen: boolean;
    onSpace: RailProps['onSpace'];
}) {
    const words = SPACE_WORDS[space];
    const soon = !opensNow(space);
    const tip = `rail-tip-${space}`;
    return (
        <span className={styles.slot}>
            <button
                aria-describedby={soon ? tip : undefined}
                aria-disabled={soon ? true : undefined}
                aria-pressed={chosen}
                className={`${styles.segment} ${soon ? styles.soon : ''}`}
                data-space={space}
                onClick={() => {
                    if (!soon) {
                        onSpace(space);
                    }
                }}
                type="button"
            >
                <span className={styles.mark}>
                    <Icon name={SPACE_ICONS[space]} />
                </span>
                <span className={styles.label}>{words.name}</span>
                {soon ? <span className={styles.soonWord}>{SOON_WORD.toLowerCase()}</span> : null}
            </button>
            {soon ? (
                <span className={styles.tip} id={tip} role="tooltip">
                    <span className={styles.tipHead}>
                        {words.name}
                        <span className={styles.pill}>{SOON_WORD}</span>
                        <kbd className={styles.tipKeys}>{SPACE_KEYS[space]}</kbd>
                    </span>
                    <span>{words.line}</span>
                </span>
            ) : null}
        </span>
    );
}

function Spaces({ route, onSpace, quiet }: Pick<RailProps, 'onSpace' | 'quiet' | 'route'>) {
    const active = !quiet && inSpace(route);
    return (
        <div
            aria-label="Spaces"
            className={`${styles.well} ${active ? styles.active : ''}`}
            role="group"
        >
            {SPACES.map((space) => (
                <Segment chosen={space === 'machine'} key={space} onSpace={onSpace} space={space} />
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
    const inner = (
        <>
            <span aria-hidden={go === undefined ? undefined : true} className={styles.dot}>
                <StateDot tone={dot.tone} word={dot.word} />
            </span>
            <span aria-hidden="true" className={styles.label}>
                Engine
            </span>
            <span aria-hidden="true" className={styles.version}>
                {below}
            </span>
            <span className={styles.hidden}>{said}</span>
        </>
    );
    return go === undefined ? (
        <div className={`${styles.place} ${styles.engine}`} title={said}>
            {inner}
        </div>
    ) : (
        <button
            className={`${styles.place} ${styles.engine}`}
            onClick={go}
            title={said}
            type="button"
        >
            {inner}
        </button>
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

// invariant: the rail is the window's, whatever the space: Home, Inbox and Settings are places of the window, the
// invariant: spaces one choice of three; the space's sidebar stays beside it on every page.
export function Rail({ route, go, onSpace, needs, engine, quiet }: RailProps) {
    const at = (page: 'home' | 'inbox' | 'settings') => ({
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
                <Place {...at('home')} icon="home" keys="Meta+0" label="Home" />
                <Place
                    {...at('inbox')}
                    count={quiet ? undefined : owedOf(needs)}
                    icon="inbox"
                    keys="Meta+I"
                    label="Inbox"
                />
            </div>
            <span aria-hidden="true" className={styles.rule} />
            <Spaces onSpace={onSpace} quiet={quiet} route={route} />
            <div className={styles.foot}>
                <Place {...at('settings')} icon="gear" keys="Meta+," label="Settings" />
                {quiet ? <QuietEngine /> : <Engine engine={engine} go={go} />}
            </div>
        </nav>
    );
}
