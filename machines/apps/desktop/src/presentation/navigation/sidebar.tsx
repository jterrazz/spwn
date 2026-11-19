import type { EngineReadOutput, MachinesMacView, OverviewNeed } from '@jterrazz/station-api';

import { machineNamesOf } from '../../domain/machines/machines';
import { needsYou } from '../../domain/overview/upkeep';
import { SEPARATOR } from '../../domain/separator';
import { useCore, useReach } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { identityOf, MachineIcon, NavItem, StateDot } from '../ui/index';
import { failedState } from '../words/failed';
import { machineKindOf } from '../words/provisional';
import { coreState, engineState, macState, ownState, vmState } from '../words/shown';
import { CORE_FOOT_WORDS, CORE_MACHINES_WORDS } from '../words/words';
import type { Carried } from './carried';
import { onListKey } from './keys';
import { ReleaseRow } from './release-row';
import { machineRoute } from './route';
import type { PlainPage, Route } from './route';

import styles from './sidebar.module.css';

export type SidebarProps = {
    carried: Carried;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    engine: Reading<EngineReadOutput>;
    route: Route;
    go: (to: Route) => void;
    onFind: () => void;
    quiet?: boolean;
};

type Place = (page: PlainPage) => { current: boolean; disabled: boolean; onSelect: () => void };

function placeOf(route: Route, go: SidebarProps['go'], quiet: boolean): Place {
    return (page) => ({
        current: quiet ? page === 'home' : route.page === page,
        disabled: quiet && page !== 'home',
        onSelect: () => {
            go({ page });
        },
    });
}

function QuietMachines() {
    const { state } = useCore();
    return (
        <div aria-labelledby="side-machines" className={styles.group} role="group">
            <h2 className={styles.caption} id="side-machines">
                Machines
            </h2>
            <p className={styles.none}>{CORE_MACHINES_WORDS[state.kind]}</p>
        </div>
    );
}

function QuietFoot() {
    const { state } = useCore();
    return (
        <div className={styles.foot}>
            <StateDot
                {...(state.kind === 'installed' ? ownState('not-started') : coreState(state.kind))}
            />
            <span className={styles.footText}>{CORE_FOOT_WORDS[state.kind]}</span>
        </div>
    );
}

function Places({ needs, place }: { needs: readonly OverviewNeed[]; place: Place }) {
    const owed = needsYou(needs);
    return (
        <div className={styles.group}>
            <NavItem {...place('home')} icon="home" label="Home" />
            <NavItem
                {...place('inbox')}
                count={
                    owed.length > 0
                        ? { urgent: owed.some((need) => need.Broken), value: owed.length }
                        : undefined
                }
                icon="inbox"
                label="Inbox"
            />
            <NavItem {...place('machine-files')} icon="file" label="Machine files" />
            <NavItem {...place('passwords')} icon="key" label="Passwords" />
        </div>
    );
}

function Machines({ macs, route, go }: Pick<SidebarProps, 'go' | 'macs' | 'route'>) {
    const on = (machine: string) => route.page === 'machine' && route.machine === machine;
    const listed = machineNamesOf(macs);
    return (
        <div aria-labelledby="side-machines" className={styles.group} role="group">
            <h2 className={styles.caption} id="side-machines">
                Machines
            </h2>
            {macs.map((mac) => (
                <div className={styles.group} key={mac.Name}>
                    <NavItem
                        current={on(mac.Name)}
                        label={mac.Name}
                        lead={<MachineIcon kind="mac" />}
                        onSelect={() => {
                            go(machineRoute(mac.Name));
                        }}
                        state={macState(mac)}
                    />
                    {(mac.VMs ?? []).map((vm) => (
                        <NavItem
                            current={on(vm.Name)}
                            indent
                            key={vm.Name}
                            label={vm.Name}
                            lead={
                                <MachineIcon
                                    identity={identityOf(vm.Name, listed)}
                                    kind={machineKindOf(vm)}
                                />
                            }
                            onSelect={() => {
                                go(machineRoute(vm.Name));
                            }}
                            state={vmState(vm)}
                        />
                    ))}
                </div>
            ))}
        </div>
    );
}

function Storage({ place }: { place: Place }) {
    return (
        <div className={styles.pinned}>
            <NavItem {...place('storage')} icon="disc" label="Storage" />
        </div>
    );
}

function countOf(value: number | undefined) {
    return value === undefined ? undefined : { urgent: false, value };
}

function Catalog({ place, carried }: { place: Place; carried: Carried }) {
    return (
        <div aria-labelledby="side-catalog" className={styles.group} role="group">
            <h2 className={styles.caption} id="side-catalog">
                Catalog
            </h2>
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
                label="Settings"
            />
        </div>
    );
}

function Engine({ engine }: Pick<SidebarProps, 'engine'>) {
    const reach = useReach();
    if (reach.kind === 'unanswered') {
        return (
            <div className={styles.foot}>
                <StateDot {...ownState('engine-absent')} />
                <span className={styles.footText}>{reach.said}</span>
            </div>
        );
    }
    if (reach.kind === 'connecting') {
        const { tone, word } = ownState('connecting');
        return (
            <div className={styles.foot}>
                <StateDot tone={tone} word={word} />
                <span className={styles.footText}>Engine {word}</span>
            </div>
        );
    }
    if (engine.kind === 'failed') {
        const { tone, word } = failedState(engine.error);
        return (
            <div className={styles.foot}>
                <StateDot tone={tone} word={word} />
                <span className={styles.footText}>{engine.error.message}</span>
            </div>
        );
    }
    if (engine.kind === 'reading') {
        return (
            <div className={styles.foot}>
                <StateDot {...ownState('reading')} />
                <span className={styles.footText}>Engine</span>
            </div>
        );
    }
    const { Version } = engine.out;
    const { tone, word } = engineState(engine.out);
    return (
        <div className={styles.foot}>
            <StateDot tone={tone} word={word} />
            <span className={styles.footText}>
                {Version === '' ? `Engine ${word}` : `Engine ${word}${SEPARATOR}${Version}`}
            </span>
        </div>
    );
}

const UNCARRIED: Carried = { apps: undefined, settings: undefined, skills: undefined };

export function Sidebar({
    carried,
    macs,
    needs,
    engine,
    route,
    go,
    onFind,
    quiet = false,
}: SidebarProps) {
    const place = placeOf(route, go, quiet);
    return (
        <nav aria-label="Sidebar" className={styles.sidebar} onKeyDown={onListKey}>
            <div className={styles.lights} data-tauri-drag-region="deep" />
            <button
                aria-keyshortcuts="Meta+K"
                aria-label="Find"
                className={styles.find}
                disabled={quiet}
                onClick={onFind}
                type="button"
            >
                <span>Find</span>
                <kbd aria-hidden="true">⌘K</kbd>
            </button>
            <div className={styles.list}>
                <Places needs={quiet ? [] : needs} place={place} />
                {quiet ? <QuietMachines /> : <Machines go={go} macs={macs} route={route} />}
                <Catalog carried={quiet ? UNCARRIED : carried} place={place} />
            </div>
            <Storage place={place} />
            {quiet ? null : <ReleaseRow go={go} />}
            {quiet ? <QuietFoot /> : <Engine engine={engine} />}
        </nav>
    );
}
