import type { MachinesMacView, OverviewReadOutput } from '@jterrazz/station-api';

import { needsYou } from '../../../domain/overview/upkeep';
import { useReach } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { LastNight } from '../../features/journal/changes';
import { Needs } from '../../features/overview/needs-panel';
import { HeldOutcomes } from '../../kit/gestures';
import { Absent, Reading as ReadingVerdict, SummaryHead, Unanswered } from '../../kit/readings';
import type { Route } from '../../navigation/route';
import { newMachineLeads } from '../../navigation/toolbar-action';
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

export function HomePage({ macs, overview, go, onNewMachine, onAddMac }: HomeProps) {
    const reach = useReach();
    if (reach.kind === 'unanswered' && overview.kind !== 'read') {
        return (
            <Page>
                <Absent reach={reach} />
            </Page>
        );
    }
    if (overview.kind === 'failed') {
        return (
            <Page>
                <Unanswered error={overview.error} size="page" />
            </Page>
        );
    }
    if (overview.kind === 'reading') {
        return (
            <Page>
                <ReadingVerdict size="page" what="This Mac" />
            </Page>
        );
    }
    const { Summary } = overview.out;
    const thisMac = macs.find((mac) => mac.ThisMac);
    return (
        <Page>
            <SummaryHead size="page" summary={Summary} verdict={Summary.Verdict} />
            {thisMac ? <HomeTiles mac={thisMac} /> : null}
            <StartHere go={go} macs={macs} onNewMachine={onNewMachine} />
            <HeldOutcomes>
                <Needs
                    go={go}
                    macs={macs}
                    needs={needsYou(Summary.Needs ?? [])}
                    standsBack={newMachineLeads(macs)}
                    title="Needs you"
                />
            </HeldOutcomes>
            {macs.length > 0 ? <HomeFleet go={go} macs={macs} onAddMac={onAddMac} /> : null}
            {holdsNoMachine(macs) ? null : <LastNight go={go} macs={macs} />}
        </Page>
    );
}
