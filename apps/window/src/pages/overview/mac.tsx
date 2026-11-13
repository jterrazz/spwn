import { Icon } from '../../design/icon';
import { identityOf, MachineIcon, MissingMark, Panel, Row, Tile } from '../../design/index';
import { machineKindOf } from '../../engine/provisional';
import { useRead } from '../../engine/react';
import { vmAbout, vmState } from '../../engine/shown';
import { sentence } from '../../model/format';
import { resourcesOf, targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import { machineRoute } from '../../shell/route';
import type { Route } from '../../shell/route';
import { NotGiven, Unanswered } from '../kit/readings';

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

export function MemoryOfMachines() {
    return <NotGiven title="Memory" what="What each machine holds of this Mac's memory." />;
}

export function Machines({ found, go }: { found: Found; go: Go }) {
    const vms = found.mac.VMs ?? [];
    if (vms.length === 0) {
        return null;
    }
    return (
        <Panel title="Its machines">
            {vms.map((vm) => (
                <Row
                    key={vm.Name}
                    lead={<MachineIcon identity={identityOf(vm.Name)} kind={machineKindOf(vm)} />}
                    onOpen={() => {
                        go(machineRoute(vm.Name));
                    }}
                    sentence={sentence([vmAbout(vm).lifetime, resourcesOf(vm)])}
                    state={vmState(vm)}
                    title={vm.Name}
                />
            ))}
        </Panel>
    );
}

export function Follows({ found, go }: { found: Found; go: Go }) {
    const drift = useRead('machinefile.drift', {}, targetOf(found));
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
                lead={<Icon name="file" />}
                onOpen={() => {
                    go({ page: 'machine-files' });
                }}
                sentence="its apps, settings and the values you set are read in Machine files"
                title={`Follows ${drift.out.Plan.Name}`}
                trail={<MissingMark />}
            />
        </Panel>
    );
}
