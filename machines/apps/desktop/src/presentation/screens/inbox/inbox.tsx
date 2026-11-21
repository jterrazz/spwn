import type { MachinesMacView, OverviewNeed, OverviewReadOutput } from '@jterrazz/station-api';
import { useState } from 'react';

import { canWait, needsYou } from '../../../domain/overview/upkeep';
import { useReach } from '../../dependencies/dependencies';
import type { Reading } from '../../dependencies/dependencies';
import { History } from '../../features/journal/changes';
import { needsOn } from '../../features/overview/needs';
import { Needs } from '../../features/overview/needs-panel';
import { HeldOutcomes } from '../../kit/gestures';
import { Absent, Reading as ReadingVerdict, SummaryHead, Unanswered } from '../../kit/readings';
import type { Route } from '../../navigation/route';
import { Switch } from '../../ui/index';
import { Page } from '../../ui/page';
import { wordOf } from '../../words/engine-words';

type Go = (to: Route) => void;

type InboxProps = {
    macs: readonly MachinesMacView[];
    overview: Reading<OverviewReadOutput>;
    go: Go;
};

function CanWait({
    needs,
    macs,
    go,
}: Pick<InboxProps, 'go' | 'macs'> & { needs: readonly OverviewNeed[] }) {
    return (
        <Needs
            go={go}
            macs={macs}
            needs={needs}
            none="Nothing can wait."
            title={wordOf('OverviewUpkeep', 'can-wait').Word}
        />
    );
}

export function InboxPage({ macs, overview, go }: InboxProps) {
    const reach = useReach();
    const [chosen, setChosen] = useState<null | string>(null);
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
                <ReadingVerdict size="page" what="what needs you" />
            </Page>
        );
    }
    const { Summary } = overview.out;
    const listed = needsOn(Summary.Needs ?? [], macs, chosen);
    const none =
        chosen === null
            ? 'No need is listed.'
            : `None of the needs listed concerns ${chosen} or its machines.`;
    return (
        <Page>
            <SummaryHead size="page" summary={Summary} verdict={Summary.Verdict} />
            <Switch
                all="All machines"
                label="Which machines"
                onChange={setChosen}
                options={macs.map((mac) => ({ id: mac.Name, label: mac.Name }))}
                value={chosen}
            />
            <HeldOutcomes>
                <Needs
                    go={go}
                    macs={macs}
                    needs={needsYou(listed)}
                    none={none}
                    note="a gesture is due"
                    title={wordOf('OverviewUpkeep', 'needs-you').Word}
                />
                <CanWait go={go} macs={macs} needs={canWait(listed)} />
            </HeldOutcomes>
            <History chosen={chosen} go={go} macs={macs} />
        </Page>
    );
}
