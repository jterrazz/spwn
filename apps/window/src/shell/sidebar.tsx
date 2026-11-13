import type { EngineReadOutput, MachinesMacView, OverviewNeed } from '@jterrazz/station-api';

import { identityOf, MachineIcon, NavItem, StateDot } from '../design/index';
import { failedState } from '../engine/failed';
import { inboxIsUrgent, machineKindOf } from '../engine/provisional';
import { useReach } from '../engine/react';
import type { Reading } from '../engine/react';
import { engineState, macState, ownState, vmState } from '../engine/shown';
import { onListKey } from './keys';
import { machineRoute } from './route';
import type { PlainPage, Route } from './route';

import styles from './sidebar.module.css';

export type SidebarProps = {
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    engine: Reading<EngineReadOutput>;
    route: Route;
    go: (to: Route) => void;
    onFind: () => void;
};

type Place = (page: PlainPage) => { current: boolean; onSelect: () => void };

function placeOf(route: Route, go: SidebarProps['go']): Place {
    return (page) => ({
        current: route.page === page,
        onSelect: () => {
            go({ page });
        },
    });
}

function Places({ needs, place }: { needs: readonly OverviewNeed[]; place: Place }) {
    return (
        <div className={styles.group}>
            <NavItem {...place('home')} icon="home" label="Home" />
            <NavItem
                {...place('inbox')}
                count={
                    needs.length > 0
                        ? { urgent: inboxIsUrgent(needs), value: needs.length }
                        : undefined
                }
                icon="inbox"
                label="Inbox"
            />
            <NavItem {...place('storage')} icon="disc" label="Storage" />
        </div>
    );
}

function Machines({ macs, route, go }: Pick<SidebarProps, 'go' | 'macs' | 'route'>) {
    const on = (machine: string) => route.page === 'machine' && route.machine === machine;
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
                                    identity={identityOf(vm.Name)}
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

function MachineFiles({ place }: { place: Place }) {
    return (
        <div className={styles.group}>
            <NavItem {...place('machine-files')} icon="file" label="Machine files" />
        </div>
    );
}

function Catalog({ place }: { place: Place }) {
    return (
        <div aria-labelledby="side-catalog" className={styles.group} role="group">
            <h2 className={styles.caption} id="side-catalog">
                Catalog
            </h2>
            <NavItem {...place('catalog-apps')} icon="apps" label="Apps" />
            <NavItem {...place('catalog-skills')} icon="skills" label="Skills" />
            <NavItem {...place('catalog-settings')} icon="settings" label="Settings" />
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
                <span className={styles.footText}>Engine {word.toLowerCase()}</span>
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
                {Version === '' ? `Engine ${word}` : `Engine ${word} · ${Version}`}
            </span>
        </div>
    );
}

export function Sidebar({ macs, needs, engine, route, go, onFind }: SidebarProps) {
    const place = placeOf(route, go);
    return (
        <nav aria-label="Sidebar" className={styles.sidebar} onKeyDown={onListKey}>
            <div className={styles.lights} data-tauri-drag-region="deep" />
            <button
                aria-keyshortcuts="Meta+K"
                aria-label="Find"
                className={styles.find}
                onClick={onFind}
                type="button"
            >
                <span>Find</span>
                <kbd aria-hidden="true">⌘K</kbd>
            </button>
            <Places needs={needs} place={place} />
            <Machines go={go} macs={macs} route={route} />
            <MachineFiles place={place} />
            <Catalog place={place} />
            <Engine engine={engine} />
        </nav>
    );
}
