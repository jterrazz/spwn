import type { ActivityReport } from '@jterrazz/station-api';
import { useState } from 'react';

import { Switch } from '../../design/index';
import { THIS_MAC_WORD } from '../../engine/words';
import type { Found } from '../../model/machines';
import type { Route } from '../../shell/route';
import { Reading, Unanswered } from '../kit/readings';
import { Page } from '../page';
import { Busiest } from './busiest';
import { Cards } from './cards';
import { Findings, Head } from './head';
import { Log } from './log';
import { MacFacts } from './mac-facts';
import { SPANS } from './measures';
import type { Span } from './measures';
import { useActivity, useHistory } from './reading';

import styles from './activity.module.css';

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
    const { readers, shown } = useActivity(found, span);
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
        now = (
            <>
                <Head report={shown.out} />
                {switcher}
                <Findings findings={shown.out.findings ?? []} />
                <Measured found={found} go={go} report={shown.out} span={span} />
            </>
        );
    } else {
        now = (
            <>
                <Reading size="tab" what={`the activity of ${name}`} />
                {switcher}
            </>
        );
    }
    return (
        <Page>
            {readers}
            {now}
            <Log found={found} />
        </Page>
    );
}
