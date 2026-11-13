import type { MachinesMacView, MachinesVM, OverviewReadOutput } from '@jterrazz/station-api';

import type { Reading as Answered } from '../../engine/entry';
import { machineKindOf, vmStateOf } from '../../engine/provisional';
import { useWatch } from '../../engine/react';
import { birthSentence } from '../../engine/shown';
import { targetOf } from '../../model/machines';
import type { Found } from '../../model/machines';
import type { Route } from '../../shell/route';
import { ActivityTiles } from '../activity/tiles';
import { DiskTile } from '../disk/disk-tile';
import { MemoryPanel } from '../home/memory';
import { HeldOutcomes } from '../kit/gestures';
import { NoVerdict, Reading, Unanswered } from '../kit/readings';
import { DeleteGesture } from '../machine/gestures';
import { Needs } from '../needs/needs-panel';
import { Page } from '../page';
import { CheckLatency, CheckTile, Health, LastDays } from './health';
import { FileChapters, ItsFile } from './its-file';
import { Follows, Machines, MachinesTile } from './mac';
import { SharedAccounts, SharedPorts } from './shared';
import { BusiestNow, WhyItExists } from './why';

import styles from './overview.module.css';

type Go = (to: Route) => void;

type Answer = Answered<OverviewReadOutput>;

function Head({ overview, name, birth }: { overview: Answer; name: string; birth?: string }) {
    if (birth !== undefined) {
        return <NoVerdict line={birth} size="tab" />;
    }
    if (overview.kind === 'failed') {
        return <Unanswered error={overview.error} size="tab" />;
    }
    if (overview.kind === 'reading') {
        return <Reading size="tab" what={name} />;
    }
    return <NoVerdict size="tab" />;
}

type NeedsOfProps = {
    overview: Answer;
    mac: MachinesMacView;
    name: string;
    title: string;
    go: Go;
};

function NeedsOf({ overview, mac, name, title, go }: NeedsOfProps) {
    if (overview.kind !== 'read') {
        return null;
    }
    return (
        <Needs
            go={go}
            macs={[mac]}
            needs={overview.out.Summary.Needs ?? []}
            note=""
            on={name}
            title={title}
        />
    );
}

function MacOverview({ found, go }: { found: Found; go: Go }) {
    const name = found.mac.Name;
    const overview = useWatch('overview.read', {}, targetOf(found));
    const up = overview.kind === 'read' ? overview.out.Summary.Head.Up : 0;
    return (
        <Page>
            <Head name={name} overview={overview} />
            <ActivityTiles
                disk={<DiskTile label="Disk" shows="used" target={targetOf(found)} />}
                found={found}
                fourth={<MachinesTile found={found} />}
                up={up}
            />
            <HeldOutcomes>
                <NeedsOf
                    go={go}
                    mac={found.mac}
                    name={name}
                    overview={overview}
                    title="Needs you, on this Mac"
                />
            </HeldOutcomes>
            <MemoryPanel macs={[found.mac]} only={name} />
            <Machines found={found} go={go} />
            <Follows found={found} go={go} />
        </Page>
    );
}

type VmProps = { mac: MachinesMacView; vm: MachinesVM; go: Go };

function StoppedOverview({ mac, vm, go }: VmProps) {
    return (
        <Page>
            <NoVerdict size="tab" />
            <FileChapters drift={undefined} go={go} vm={vm.Name} />
            <DeleteGesture go={go} key={vm.Name} mac={mac} vm={vm} />
        </Page>
    );
}

function VmOverview({ mac, vm, go }: VmProps) {
    const found = { mac, vm };
    const overview = useWatch('overview.read', {}, targetOf(found));
    const up = overview.kind === 'read' ? overview.out.Summary.Head.Up : 0;
    const server = machineKindOf(vm) === 'server';
    return (
        <Page>
            {vm.Birth === null ? (
                <Head name={vm.Name} overview={overview} />
            ) : (
                <Head birth={birthSentence(vm.Birth)} name={vm.Name} overview={overview} />
            )}
            <ActivityTiles
                disk={<DiskTile label="Disk" shows="used" target={targetOf(found)} />}
                found={found}
                fourth={server ? <CheckTile vm={vm} /> : undefined}
                up={up}
            />
            <div className={styles.columns}>
                <div className={styles.column}>
                    <HeldOutcomes>
                        {server ? (
                            <>
                                <Health vm={vm} />
                                <CheckLatency vm={vm} />
                                <LastDays vm={vm} />
                            </>
                        ) : (
                            <>
                                <ItsFile found={found} go={go} />
                                <SharedPorts found={found} go={go} vm={vm.Name} />
                                <SharedAccounts found={found} go={go} vm={vm.Name} />
                            </>
                        )}
                    </HeldOutcomes>
                </div>
                <div className={styles.column}>
                    <WhyItExists vm={vm} />
                    <BusiestNow found={found} />
                    <DeleteGesture go={go} key={vm.Name} mac={mac} vm={vm} />
                </div>
            </div>
        </Page>
    );
}

export function OverviewTab({ found, go }: { found: Found; go: Go }) {
    const { mac, vm } = found;
    if (!vm) {
        return <MacOverview found={found} go={go} />;
    }
    if (vmStateOf(vm) === 'stopped') {
        return <StoppedOverview go={go} key={vm.Name} mac={mac} vm={vm} />;
    }
    return <VmOverview go={go} key={vm.Name} mac={mac} vm={vm} />;
}
