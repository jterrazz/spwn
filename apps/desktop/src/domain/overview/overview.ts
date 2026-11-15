import type { OverviewLine, OverviewSummary } from '@jterrazz/station-api';

export function lineOf(
    summary: OverviewSummary,
    section: OverviewLine['Section'],
): OverviewLine | undefined {
    return (summary.Lines ?? []).find((line) => line.Section === section);
}
