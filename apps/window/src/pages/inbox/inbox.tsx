import type { MachinesMacView, OverviewReadOutput } from '@jterrazz/station-api';
import { useState } from 'react';

import { Switch } from '../../design/index';
import { useReach } from '../../engine/react';
import type { Reading } from '../../engine/react';
import type { Route } from '../../shell/route';
import { History } from '../changes/changes';
import { HeldOutcomes } from '../kit/gestures';
import {
    Absent,
    NotGiven,
    NoVerdict,
    Reading as ReadingVerdict,
    Unanswered,
} from '../kit/readings';
import { needsOn } from '../needs/needs';
import { Needs } from '../needs/needs-panel';
import { Page } from '../page';

type Go = (to: Route) => void;

type InboxProps = {
    macs: readonly MachinesMacView[];
    overview: Reading<OverviewReadOutput>;
    go: Go;
};

export function InboxPage({ macs, overview, go }: InboxProps) {
    const reach = useReach();
    const [chosen, setChosen] = useState<null | string>(null);
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
                <ReadingVerdict size="page" what="what needs you" />
            </Page>
        );
    }
    const { Summary } = overview.out;
    const needs = needsOn(Summary.Needs ?? [], macs, chosen);
    const none =
        chosen === null
            ? 'No need is listed.'
            : `None of the needs listed concerns ${chosen} or its machines.`;
    return (
        <Page>
            <NoVerdict size="page" />
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
                    needs={needs}
                    none={none}
                    note="a gesture is due"
                    title="Needs you"
                />
            </HeldOutcomes>
            <NotGiven title="Can wait" what="What can wait, apart from what needs you" />
            <History chosen={chosen} go={go} macs={macs} />
        </Page>
    );
}
