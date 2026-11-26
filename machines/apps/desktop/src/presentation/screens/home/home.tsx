import type { MachinesMacView, OverviewReadOutput } from '@jterrazz/station-api';

import { needsYou } from '../../../domain/overview/upkeep';
import { useReach } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { LastNight } from '../../features/journal/changes';
import { Needs } from '../../features/overview/needs-panel';
import { HeldOutcomes } from '../../kit/gestures';
import { Absent, SummaryHead, Unanswered } from '../../kit/readings';
import { SkeletonHead } from '../../kit/skeleton';
import { newMachineLeads } from '../../navigation/bar-action';
import type { Route } from '../../navigation/route';
import { Page } from '../../ui/page';
import { HomeFleet } from './home-fleet';
import { HomeTiles } from './home-tiles';
import { holdsNoMachine, StartHere } from './start-here';

type Go = (to: Route) => void;

export type HomeProps = {
    macs: readonly MachinesMacView[];
    overview: Reading<OverviewReadOutput>;
    go: Go;
    onNewMachine?: (() => void) | undefined;
    onAddMac?: (() => void) | undefined;
};

type Summary = OverviewReadOutput['Summary'];

// value: while the summary is read, the head is drawn as bones and the rest of Home, read apart, is drawn already.
function Head({ summary }: { summary: Summary | undefined }) {
    if (summary === undefined) {
        return <SkeletonHead size="page" what="This Mac" />;
    }
    return <SummaryHead size="page" summary={summary} verdict={summary.Verdict} />;
}

function HomeNeeds({ summary, macs, go }: { summary: Summary; macs: HomeProps['macs']; go: Go }) {
    return (
        <HeldOutcomes>
            <Needs
                go={go}
                macs={macs}
                needs={needsYou(summary.Needs ?? [])}
                standsBack={newMachineLeads(macs, summary.Needs ?? [])}
                title="Needs you"
            />
        </HeldOutcomes>
    );
}

export function HomePage({ macs, overview, go, onNewMachine, onAddMac }: HomeProps) {
    const reach = useReach();
    if (reach.kind === 'unanswered' && overview.kind !== 'read') {
        return (
            <Page density="comfortable">
                <Absent reach={reach} />
            </Page>
        );
    }
    if (overview.kind === 'failed') {
        return (
            <Page density="comfortable">
                <Unanswered error={overview.error} size="page" />
            </Page>
        );
    }
    const summary = overview.kind === 'read' ? overview.out.Summary : undefined;
    const thisMac = macs.find((mac) => mac.ThisMac);
    return (
        <Page density="comfortable">
            <Head summary={summary} />
            {thisMac ? <HomeTiles mac={thisMac} /> : null}
            <StartHere go={go} macs={macs} onNewMachine={onNewMachine} />
            {summary === undefined ? null : <HomeNeeds go={go} macs={macs} summary={summary} />}
            {macs.length > 0 ? <HomeFleet go={go} macs={macs} onAddMac={onAddMac} /> : null}
            {holdsNoMachine(macs) ? null : <LastNight go={go} macs={macs} />}
        </Page>
    );
}
