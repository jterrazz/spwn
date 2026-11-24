import type { MachinesMacView } from '@jterrazz/station-api';

import { machineNamesOf } from '../../domain/machines/machines';
import { useCore, useReach } from '../dependencies/dependencies';
import { IconButton, identityOf, MachineIcon, NavItem } from '../ui/index';
import { FIT_MACHINES_WORDS } from '../words/fit';
import { machineKindOf } from '../words/provisional';
import { macState, vmState } from '../words/shown';
import { CORE_MACHINES_WORDS, NEW_MACHINE_WORD } from '../words/words';
import type { Carried } from './carried';
import { onListKey } from './keys';
import { ReleaseRow } from './release-row';
import { machineRoute } from './route';
import type { PlainPage, Route } from './route';
import { SPACE_WORDS } from './spaces';

import styles from './sidebar.module.css';

export type SidebarProps = {
    carried: Carried;
    macs: readonly MachinesMacView[];
    route: Route;
    go: (to: Route) => void;
    onFind: () => void;
    onNewMachine: () => void;
    quiet?: boolean;
};

type Place = (page: PlainPage) => { current: boolean; disabled: boolean; onSelect: () => void };

function placeOf(route: Route, go: SidebarProps['go'], quiet: boolean): Place {
    return (page) => ({
        current: !quiet && route.page === page,
        disabled: quiet,
        onSelect: () => {
            go({ page });
        },
    });
}

function QuietMachines() {
    const { state } = useCore();
    const reach = useReach();
    return (
        <div aria-labelledby="side-machines" className={styles.group} role="group">
            <h2 className={styles.caption} id="side-machines">
                Machines
            </h2>
            <p className={styles.none}>
                {reach.kind === 'incompatible'
                    ? FIT_MACHINES_WORDS[reach.fit]
                    : CORE_MACHINES_WORDS[state.kind]}
            </p>
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
                label="Machine settings"
            />
            <NavItem {...place('machine-files')} icon="file" label="Machine files" />
        </div>
    );
}

// invariant: Storage and Engine head the sidebar, above the machines: each shows every Mac at once.
function Fleet({ place }: { place: Place }) {
    return (
        <div aria-label="Every Mac" className={styles.group} role="group">
            <NavItem {...place('storage')} icon="disc" label="Storage" />
            <NavItem {...place('engine')} icon="engine" label="Engine" />
        </div>
    );
}

const UNCARRIED: Carried = { apps: undefined, settings: undefined, skills: undefined };

export function Sidebar(props: SidebarProps) {
    const { carried, macs, route, go, onFind, onNewMachine, quiet = false } = props;
    const place = placeOf(route, go, quiet);
    return (
        <nav aria-label="Sidebar" className={styles.sidebar} onKeyDown={onListKey}>
            <div className={styles.lights} data-tauri-drag-region="deep">
                <h2 className={styles.space}>{SPACE_WORDS.machine.name}</h2>
                <IconButton
                    disabled={quiet}
                    icon="plus"
                    label={NEW_MACHINE_WORD}
                    onClick={onNewMachine}
                />
            </div>
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
                <Fleet place={place} />
                {quiet ? <QuietMachines /> : <Machines go={go} macs={macs} route={route} />}
                <Catalog carried={quiet ? UNCARRIED : carried} place={place} />
            </div>
            {quiet ? null : <ReleaseRow go={go} />}
        </nav>
    );
}
