import type { MachinesMacView, OverviewReadOutput } from '@jterrazz/station-api';

import { useReach } from '../../engine/react';
import type { Reading } from '../../engine/react';
import type { Route } from '../../shell/route';
import { HeldOutcomes } from '../kit/gestures';
import {
    Absent,
    NotGiven,
    NoVerdict,
    Reading as ReadingVerdict,
    Unanswered,
} from '../kit/readings';
import { Needs } from '../needs/needs-panel';
import { Page } from '../page';
import { HomeMachines } from './home-machines';
import { HomeTiles } from './home-tiles';
import { MemoryPanel } from './memory';
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
    if (reach.kind === 'unanswered') {
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
            <NoVerdict size="page" />
            {thisMac ? <HomeTiles mac={thisMac} /> : null}
            <StartHere go={go} macs={macs} onNewMachine={onNewMachine} />
            <HeldOutcomes>
                <Needs go={go} macs={macs} needs={Summary.Needs ?? []} title="Needs you" />
            </HeldOutcomes>
            <MemoryPanel macs={macs} />
            {macs.length > 0 ? <HomeMachines go={go} macs={macs} onAddMac={onAddMac} /> : null}
            {holdsNoMachine(macs) ? null : (
                <NotGiven title="Last night" what="What moved, held or failed overnight" />
            )}
        </Page>
    );
}
