import type { MachinesVM } from '@jterrazz/station-api';

import { duration, sentence } from '../../../../domain/format';
import { driftAsked } from '../../../../domain/machines/machines';
import type { Found } from '../../../../domain/machines/machines';
import { between, firstUnitOf } from '../../../../domain/when';
import { useRead, useWatch } from '../../../dependencies/dependencies';
import { useIdentityOf } from '../../../kit/identity';
import { Unanswered } from '../../../kit/readings';
import { machineRoute } from '../../../navigation/route';
import type { Route } from '../../../navigation/route';
import { Icon } from '../../../ui/icon';
import { MachineIcon, Panel, Row, Tile } from '../../../ui/index';
import { nounOf } from '../../../words/engine-words';
import { machineKindOf } from '../../../words/provisional';
import { verdictShown, vmState } from '../../../words/shown';

import styles from './overview.module.css';

type Go = (to: Route) => void;

export function MachinesTile({ found, foot }: { found: Found; foot: string | undefined }) {
    const count = (found.mac.VMs ?? []).length;
    return <Tile foot={foot} label="Its machines" unit={nounOf('machine', count)} value={count} />;
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
        return { ...shown, word: vm.Birth.sentence };
    }
    return { ...shown, asWritten: true, word: sentence([shown.word, upFor(vm, read)]) };
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
