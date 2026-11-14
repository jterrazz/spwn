import type { OverviewNeed } from '@jterrazz/station-api';

type Upkeep = OverviewNeed['Upkeep'];

export function isNeedsYou({ Upkeep }: Pick<OverviewNeed, 'Upkeep'>): boolean {
    return Upkeep === 'needs-you';
}

function withUpkeep(needs: readonly OverviewNeed[], wanted: Upkeep): readonly OverviewNeed[] {
    return needs.filter(({ Upkeep }) => Upkeep === wanted);
}

export function needsYou(needs: readonly OverviewNeed[]): readonly OverviewNeed[] {
    return withUpkeep(needs, 'needs-you');
}

export function canWait(needs: readonly OverviewNeed[]): readonly OverviewNeed[] {
    return withUpkeep(needs, 'can-wait');
}
