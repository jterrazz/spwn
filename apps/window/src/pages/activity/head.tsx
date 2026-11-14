import type { ActivityReport, CoreFindingFinding } from '@jterrazz/station-api';

import { Panel, Row, Value, Verdict } from '../../design/index';
import { engineShown, ownState } from '../../engine/shown';
import { ACTIVITY_SOURCE_WORDS, SILENT_METER_WORDS } from '../../engine/words';
import { clock, fixLine, seconds, sentence } from '../../model/format';
import { shortDay } from './when';

function Calm({ report }: { report: ActivityReport }) {
    const every = report.chart ? report.chart.every : 0;
    const since = shortDay(report.since);
    return (
        <Verdict
            line={sentence([
                'Read live',
                ACTIVITY_SOURCE_WORDS[report.source],
                every > 0 ? `a point every ${seconds(every)}` : undefined,
                since === undefined
                    ? undefined
                    : `history kept since ${since} ${clock(report.since) ?? ''}`,
            ])}
            size="tab"
            title="No finding from the engine."
            {...ownState('no-finding')}
        />
    );
}

export function Head({ report }: { report: ActivityReport }) {
    const first = (report.findings ?? [])[0];
    if (report.silent === true) {
        return (
            <Verdict
                line={SILENT_METER_WORDS.line}
                size="tab"
                title={SILENT_METER_WORDS.title}
                {...ownState('not-measured')}
            />
        );
    }
    if (!first) {
        return <Calm report={report} />;
    }
    const fix = fixLine(first.Fix);
    const severity = engineShown('CoreFindingSeverity', first.Severity);
    return (
        <Verdict
            action={fix === '' ? undefined : <Value>{fix}</Value>}
            line={sentence([severity.word, first.Subject])}
            size="tab"
            title={first.Text}
            {...severity}
        />
    );
}

export function Findings({ findings }: { findings: readonly CoreFindingFinding[] }) {
    if (findings.length < 2) {
        return null;
    }
    return (
        <Panel note={String(findings.length)} title="Findings">
            {findings.map((finding) => {
                const fix = fixLine(finding.Fix);
                return (
                    <Row
                        key={finding.Subject}
                        state={engineShown('CoreFindingSeverity', finding.Severity)}
                        stateAt="start"
                        title={finding.Text}
                        value={fix === '' ? null : <Value>{fix}</Value>}
                        valueSize="long"
                    />
                );
            })}
        </Panel>
    );
}
