import type { MachinesMacView } from '@jterrazz/station-api';

import { Button, MachineIcon, StatePill, Switch } from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import { useWatch } from '../../engine/react';
import { engineViewWord, macState, vmKindWord, vmLifetimeWord, vmState } from '../../engine/shown';
import { SECTION_WORDS, THIS_MAC_WORD } from '../../engine/words';
import { sentence } from '../../model/format';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { lineOf } from '../../model/overview';
import { SEPARATOR } from '../../separator';
import { machineRoute, TAB_PANEL, TABS } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';
import type { Opening } from '../../shell/window';
import { useIdentityOf } from '../kit/identity';
import { LifecycleGestures } from './gestures';
import { TerminalButton } from './terminal';

import styles from './machine.module.css';

type Props = {
    found: Found;
    tab: Tab;
    go: (to: Route) => void;
    onNewMachine?: ((opening: Opening) => void) | undefined;
};

function useAwakeWord(mac: MachinesMacView): string | undefined {
    const overview = useWatch('overview.read', {}, targetOf({ mac }));
    if (overview.kind !== 'read') {
        return undefined;
    }
    const line = lineOf(overview.out.Summary, 'awake');
    if (line === undefined) {
        return undefined;
    }
    const words =
        line.Unread === '' ? sentence((line.Parts ?? []).map((part) => part.Word)) : line.Unread;
    return words === '' ? undefined : `${SECTION_WORDS.awake}: ${words}`;
}

function MacLine({ mac }: { mac: MachinesMacView }) {
    const engine = engineViewWord(mac.Engine);
    const awake = useAwakeWord(mac);
    return (
        <p className={styles.sentence}>
            {sentence([
                mac.ThisMac ? THIS_MAC_WORD : undefined,
                engine === undefined ? undefined : `Engine ${engine}`,
                awake,
            ])}
        </p>
    );
}

function VmLine({ found, go }: { found: Found; go: Props['go'] }) {
    const { mac, vm } = found;
    if (!vm) {
        return null;
    }
    return (
        <p className={styles.sentence}>
            {`${sentence([vmKindWord(vm), vmLifetimeWord(vm)])}${SEPARATOR}on `}
            <button
                className={styles.link}
                onClick={() => {
                    go(machineRoute(mac.Name));
                }}
                type="button"
            >
                {mac.Name}
            </button>
        </p>
    );
}

function Actions({ found, onNewMachine }: Pick<Props, 'found' | 'onNewMachine'>) {
    const { mac, vm } = found;
    if (!vm) {
        return (
            <div className={styles.actions}>
                <Button
                    name={`New machine on ${mac.Name}`}
                    onClick={() => {
                        onNewMachine?.({ on: mac.Name });
                    }}
                >
                    New machine
                </Button>
            </div>
        );
    }
    return (
        <LifecycleGestures key={vm.Name} mac={mac} terminal={<TerminalButton vm={vm} />} vm={vm} />
    );
}

export function MachineHeader({ found, tab, go, onNewMachine }: Props) {
    const { mac, vm } = found;
    const name = vm?.Name ?? mac.Name;
    const identityOf = useIdentityOf();
    return (
        <div className={styles.header}>
            <div className={styles.inner}>
                <div className={styles.top}>
                    <div className={styles.who}>
                        <MachineIcon
                            identity={vm ? identityOf(vm.Name) : undefined}
                            kind={vm ? machineKindOf(vm) : 'mac'}
                            size="large"
                        />
                        <div>
                            <h2 className={styles.name}>
                                {name}
                                <StatePill {...(vm ? vmState(vm) : macState(mac))} />
                            </h2>
                            {vm ? <VmLine found={found} go={go} /> : <MacLine mac={mac} />}
                        </div>
                    </div>
                    <Actions found={found} onNewMachine={onNewMachine} />
                </div>
                <Switch
                    controls={TAB_PANEL}
                    label="Tabs"
                    onChange={(next) => {
                        go(machineRoute(name, next));
                    }}
                    options={TABS.map((option) => ({ id: option, label: option }))}
                    value={tab}
                    variant="tabs"
                />
            </div>
        </div>
    );
}
