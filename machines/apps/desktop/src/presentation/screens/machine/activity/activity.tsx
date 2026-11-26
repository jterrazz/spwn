import type { ActivityReport } from '@jterrazz/station-api';
import { useState } from 'react';

import { SPANS } from '../../../../domain/activity/measures';
import type { Span } from '../../../../domain/activity/measures';
import type { Found } from '../../../../domain/machines/machines';
import { Busiest } from '../../../features/activity/busiest';
import { useActivity, useHistory } from '../../../features/activity/reading';
import { Unanswered } from '../../../kit/readings';
import { Placeheld, Skeleton } from '../../../kit/skeleton';
import type { Route } from '../../../navigation/route';
import { Switch, VerdictSkeleton } from '../../../ui/index';
import { Page } from '../../../ui/page';
import { THIS_MAC_WORD } from '../../../words/words';
import { Cards } from './cards';
import { Findings, Head } from './head';
import { Log } from './log';
import { MacFacts } from './mac-facts';

import styles from '../../../features/activity/activity.module.css';

type Go = (to: Route) => void;

function Measured({
    report,
    found,
    span,
    go,
}: {
    report: ActivityReport;
    found: Found;
    span: Span;
    go: Go;
}) {
    const name = found.vm?.Name ?? found.mac.Name;
    const history = useHistory(found);
    if (report.silent === true) {
        return null;
    }
    return (
        <>
            <Cards history={history} name={name} report={report} span={span} />
            <Busiest found={found} go={go} report={report} />
            {found.vm ? null : (
                <MacFacts
                    report={report}
                    title={found.mac.ThisMac ? THIS_MAC_WORD : found.mac.Name}
                />
            )}
        </>
    );
}

export function ActivityTab({ found, go }: { found: Found; go: Go }) {
    const name = found.vm?.Name ?? found.mac.Name;
    const [span, setSpan] = useState<Span>('6m');
    const shown = useActivity(found, span);
    const switcher = (
        <div className={styles.bar}>
            <Switch
                label="Window"
                onChange={setSpan}
                options={SPANS.map((option) => ({ id: option.id, label: option.label }))}
                value={span}
            />
        </div>
    );
    let now;
    if (shown?.kind === 'failed') {
        now = <Unanswered error={shown.error} size="tab" />;
    } else if (shown) {
        const held = shown.placeholder === true;
        now = (
            <>
                <Placeheld held={held}>
                    <Head report={shown.out} />
                </Placeheld>
                {switcher}
                <Placeheld held={held}>
                    <Findings findings={shown.out.findings ?? []} />
                    <Measured found={found} go={go} report={shown.out} span={span} />
                </Placeheld>
            </>
        );
    } else {
        now = (
            <Skeleton what={`the activity of ${name}`}>
                <VerdictSkeleton size="tab" />
                {switcher}
            </Skeleton>
        );
    }
    return (
        <Page density="comfortable">
            {now}
            <Log found={found} />
        </Page>
    );
}
