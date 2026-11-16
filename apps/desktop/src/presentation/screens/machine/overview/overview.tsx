import type {
    CoreVerdictVerdict,
    MachinesMacView,
    MachinesVM,
    OverviewReadOutput,
} from '@jterrazz/station-api';

import type { Reading as Answered } from '../../../../application/use-cases/entry';
import { sentence } from '../../../../domain/format';
import { isStopped, targetOf } from '../../../../domain/machines/machines';
import type { Found } from '../../../../domain/machines/machines';
import { needsYou } from '../../../../domain/overview/upkeep';
import { useWatch } from '../../../dependencies/dependencies';
import { ActivityTiles } from '../../../features/activity/tiles';
import { DiskTile } from '../../../features/disk/disk-tile';
import { MemoryPanel } from '../../../features/machines/memory';
import { Needs } from '../../../features/overview/needs-panel';
import { HeldOutcomes } from '../../../kit/gestures';
import { NoVerdict, Reading, SummaryHead, Unanswered, VerdictHead } from '../../../kit/readings';
import type { Route } from '../../../navigation/route';
import { Page } from '../../../ui/page';
import { machineKindOf } from '../../../words/provisional';
import { DeleteGesture } from '../gestures';
import { CheckLatency, CheckTile, Health, LastDays } from './health';
import { FileChapters, ItsFile } from './its-file';
import { Follows, Machines, MachinesTile } from './mac';
import { SharedPorts, SignInsPanel } from './shared';
import { BusiestNow, WhyItExists } from './why';

import styles from './overview.module.css';

type Go = (to: Route) => void;

type Answer = Answered<OverviewReadOutput>;

type MacHeadProps = { overview: Answer; name: string; unseen?: CoreVerdictVerdict | undefined };

function MacHead({ overview, name, unseen }: MacHeadProps) {
    if (unseen !== undefined) {
        return <VerdictHead size="tab" verdict={unseen} />;
    }
    if (overview.kind === 'failed') {
        return <Unanswered error={overview.error} size="tab" />;
    }
    if (overview.kind === 'reading') {
        return <Reading size="tab" what={name} />;
    }
    const { Summary } = overview.out;
    return Summary.Head.Verdict === undefined ? (
        <NoVerdict size="tab" />
    ) : (
        <SummaryHead size="tab" summary={Summary} verdict={Summary.Head.Verdict} />
    );
}

function VmHead({ vm, overview }: { vm: MachinesVM; overview?: Answer }) {
    const birth = overview?.kind === 'read' ? overview.out.Creating?.sentence : undefined;
    if (vm.Head === undefined) {
        return <NoVerdict size="tab" {...(birth === undefined ? {} : { line: birth })} />;
    }
    const line = sentence([vm.Head.Line, birth]);
    return <VerdictHead size="tab" verdict={{ ...vm.Head, Line: line }} />;
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
            needs={needsYou(overview.out.Summary.Needs ?? [])}
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
    const foot = overview.kind === 'read' ? overview.out.Summary.Machines : undefined;
    return (
        <Page>
            <MacHead name={name} overview={overview} unseen={found.mac.Unseen} />
            <ActivityTiles
                disk={<DiskTile label="Disk" shows="used" target={targetOf(found)} />}
                found={found}
                fourth={<MachinesTile foot={foot} found={found} />}
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
            {found.mac.Unseen === undefined ? <MemoryPanel macs={[found.mac]} only={name} /> : null}
            <Machines found={found} go={go} />
            <Follows found={found} go={go} />
        </Page>
    );
}

type VmProps = { mac: MachinesMacView; vm: MachinesVM; go: Go };

function StoppedOverview({ mac, vm, go }: VmProps) {
    return (
        <Page>
            <VmHead vm={vm} />
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
            <VmHead overview={overview} vm={vm} />
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
                                <SignInsPanel found={found} go={go} vm={vm.Name} />
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
    if (isStopped(vm)) {
        return <StoppedOverview go={go} key={vm.Name} mac={mac} vm={vm} />;
    }
    return <VmOverview go={go} key={vm.Name} mac={mac} vm={vm} />;
}
