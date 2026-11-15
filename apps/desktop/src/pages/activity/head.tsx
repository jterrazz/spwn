import type { ActivityReport, CoreFindingFinding } from '@jterrazz/station-api';

import { fixLine } from '../../domain/format';
import { Panel, Row, Value } from '../../presentation/ui/index';
import { engineShown } from '../../presentation/words/shown';
import { VerdictHead } from '../kit/readings';

export function Head({ report }: { report: ActivityReport }) {
    const fix = fixLine((report.findings ?? [])[0]?.Fix);
    return (
        <VerdictHead
            action={fix === '' ? undefined : <Value>{fix}</Value>}
            size="tab"
            verdict={report.verdict}
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
