import type { MachinesMacView } from '@jterrazz/station-api';
import { useState } from 'react';

import { machineNamesOf } from '../../domain/machines/machines';
import { useCore, useReach } from '../dependencies/dependencies';
import { Icon } from '../ui/icon';
import { identityOf, MachineIcon, NavItem } from '../ui/index';
import { FIT_MACHINES_WORDS } from '../words/fit';
import { machineKindOf } from '../words/provisional';
import { macState, vmState } from '../words/shown';
import { CORE_MACHINES_WORDS } from '../words/words';
import { machineRoute } from './route';
import type { Route } from './route';

import styles from './column.module.css';

type Go = (to: Route) => void;

export function QuietMachines() {
    const { state } = useCore();
    const reach = useReach();
    return (
        <div aria-labelledby="column-machines" className={styles.group} role="group">
            <div className={styles.caption} id="column-machines">
                Machines
            </div>
            <p className={styles.none}>
                {reach.kind === 'incompatible'
                    ? FIT_MACHINES_WORDS[reach.fit]
                    : CORE_MACHINES_WORDS[state.kind]}
            </p>
        </div>
    );
}

type MacProps = {
    mac: MachinesMacView;
    listed: readonly string[];
    route: Route;
    go: Go;
};

// invariant: a Mac folds its VMs under its row, which then says how many it holds; the Mac of the open machine
// invariant: stays open, so the open machine is always in sight.
function Mac({ mac, listed, route, go }: MacProps) {
    const [folded, setFolded] = useState(false);
    const vms = mac.VMs ?? [];
    const on = (machine: string) => route.page === 'machine' && route.machine === machine;
    const holdsOpen = vms.some((vm) => on(vm.Name));
    const open = !folded || holdsOpen;
    return (
        <div className={styles.mac}>
            <div className={styles.macRow}>
                {vms.length === 0 ? (
                    <span className={styles.fold} />
                ) : (
                    <button
                        aria-expanded={open}
                        aria-label={`${mac.Name}'s machines`}
                        className={styles.fold}
                        disabled={holdsOpen}
                        onClick={() => {
                            setFolded(open);
                        }}
                        type="button"
                    >
                        <Icon name="chevron" />
                    </button>
                )}
                <NavItem
                    count={open ? undefined : { urgent: false, value: vms.length }}
                    current={on(mac.Name)}
                    label={mac.Name}
                    lead={<MachineIcon kind="mac" />}
                    onSelect={() => {
                        go(machineRoute(mac.Name));
                    }}
                    state={macState(mac)}
                />
            </div>
            {open && vms.length > 0 ? (
                <div className={styles.vms}>
                    {vms.map((vm) => (
                        <NavItem
                            current={on(vm.Name)}
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
            ) : null}
        </div>
    );
}

type MachinesProps = {
    macs: readonly MachinesMacView[];
    route: Route;
    go: Go;
};

export function Machines({ macs, route, go }: MachinesProps) {
    const listed = machineNamesOf(macs);
    return (
        <div aria-labelledby="column-machines" className={styles.group} role="group">
            <div className={styles.caption} id="column-machines">
                Machines
            </div>
            {macs.map((mac) => (
                <Mac go={go} key={mac.Name} listed={listed} mac={mac} route={route} />
            ))}
        </div>
    );
}
