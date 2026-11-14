import type { OverviewNeed } from '@jterrazz/station-api';

import { machineRoute } from '../../shell/route';
import type { Route, Tab } from '../../shell/route';

const TAB_OF_SECTION: Readonly<Record<OverviewNeed['Section'], Tab>> = {
    accounts: 'Accounts',
    activity: 'Activity',
    apps: 'Overview',
    awake: 'Overview',
    background: 'Overview',
    disk: 'Storage',
    images: 'Overview',
    overview: 'Overview',
    ports: 'Ports',
    settings: 'Overview',
    ssh: 'Overview',
};

export function tabOf({ Section }: Pick<OverviewNeed, 'Section'>): Tab {
    return TAB_OF_SECTION[Section];
}

export function routeOfNeed(need: OverviewNeed): Route {
    return machineRoute(need.Machine, tabOf(need));
}

export type Held = { Name: string; VMs: null | readonly { Name: string }[] };

export function needsOn(
    needs: readonly OverviewNeed[],
    macs: readonly Held[],
    chosen: null | string,
): readonly OverviewNeed[] {
    if (chosen === null) {
        return needs;
    }
    const mac = macs.find((candidate) => candidate.Name === chosen);
    if (!mac) {
        return [];
    }
    const names = new Set([mac.Name, ...(mac.VMs ?? []).map((vm) => vm.Name)]);
    return needs.filter((need) => names.has(need.Machine));
}

export function gestureOf(need: OverviewNeed): undefined | { call: string } | { text: string } {
    if (need.Call !== '') {
        return { call: need.Call };
    }
    const text = need.Command === '' ? need.Path : need.Command;
    return text === '' ? undefined : { text };
}
