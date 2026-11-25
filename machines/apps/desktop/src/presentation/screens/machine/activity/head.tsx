import type { ActivityReport, CoreFindingFinding } from '@jterrazz/station-api';

import { VerdictHead } from '../../../kit/readings';
import { CopyButton } from '../../../ui/copy-button';
import { Panel, Row } from '../../../ui/index';
import { engineShown } from '../../../words/shown';

export function Head({ report }: { report: ActivityReport }) {
    const command = (report.findings ?? [])[0]?.Fix.Command ?? '';
    return (
        <VerdictHead
            action={
                command === '' ? undefined : (
                    <CopyButton label="Copy the command" name="Copy the command" text={command} />
                )
            }
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
                const { Command } = finding.Fix;
                return (
                    <Row
                        key={finding.Subject}
                        state={engineShown('CoreFindingSeverity', finding.Severity)}
                        stateAt="start"
                        title={finding.Text}
                        value={
                            Command === '' ? null : (
                                <CopyButton
                                    label="Copy the command"
                                    name="Copy the command"
                                    text={Command}
                                />
                            )
                        }
                        valueSize="long"
                    />
                );
            })}
        </Panel>
    );
}
