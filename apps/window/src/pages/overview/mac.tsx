import type { MachinesVM } from '@jterrazz/station-api';

import { Icon } from '../../design/icon';
import { MachineIcon, MissingMark, Panel, Row, Tile } from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import { useRead, useWatch } from '../../engine/react';
import { birthSentence, verdictShown, vmState } from '../../engine/shown';
import { duration, sentence } from '../../model/format';
import { driftAsked } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { between, firstUnitOf } from '../activity/when';
import { useIdentityOf } from '../kit/identity';
import { Unanswered } from '../kit/readings';

import styles from './overview.module.css';

type Go = (to: Route) => void;

export function MachinesTile({ found }: { found: Found }) {
    const count = (found.mac.VMs ?? []).length;
    return (
        <Tile
            foot={<MissingMark />}
            label="Its machines"
            unit={count === 1 ? 'machine' : 'machines'}
            value={count}
        />
    );
}

// value: how long a VM has run, from its boot to the moment machines.list was read: a date drawn as "6 d".
function upFor(vm: MachinesVM, read: string | undefined): string | undefined {
    const span = between(vm.BootAt, read);
    if (span === undefined) {
        return undefined;
    }
    const { value, unit } = firstUnitOf(duration(span));
    return `up ${value} ${unit}`;
}

function stateUp(vm: MachinesVM, read: string | undefined) {
    const shown = vmState(vm);
    if (vm.Birth !== null) {
        return { ...shown, word: birthSentence(vm.Birth) };
    }
    return { ...shown, word: sentence([shown.word, upFor(vm, read)]) };
}

export function Machines({ found, go }: { found: Found; go: Go }) {
    const listing = useWatch('machines.list', {});
    const read = listing.kind === 'read' ? listing.out.All.Read : undefined;
    const vms = found.mac.VMs ?? [];
    const identityOf = useIdentityOf();
    if (vms.length === 0) {
        return null;
    }
    return (
        <Panel title="Its machines">
            {vms.map((vm) => (
                <Row
                    key={vm.Name}
                    lead={<MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} />}
                    leadSize="machine"
                    onOpen={() => {
                        go(machineRoute(vm.Name));
                    }}
                    sentence={sentence([
                        vm.Purpose,
                        vm.Creator === '' ? undefined : `by ${vm.Creator}`,
                    ])}
                    state={stateUp(vm, read)}
                    title={vm.Name}
                />
            ))}
        </Panel>
    );
}

export function Follows({ found, go }: { found: Found; go: Go }) {
    const drift = useRead('machinefile.drift', driftAsked(found));
    if (drift.kind === 'reading') {
        return null;
    }
    if (drift.kind === 'failed') {
        return (
            <Panel title="Packages">
                <Unanswered error={drift.error} size="panel" />
            </Panel>
        );
    }
    return (
        <Panel title="Packages">
            <Row
                lead={
                    <span aria-hidden="true" className={styles.fileTile}>
                        <Icon name="file" />
                    </span>
                }
                onOpen={() => {
                    go({ page: 'machine-files' });
                }}
                sentence="its apps, settings and the values you set are read in Machine files"
                state={verdictShown(drift.out.Verdict)}
                title={`Follows ${drift.out.Plan.Name}`}
            />
        </Panel>
    );
}
