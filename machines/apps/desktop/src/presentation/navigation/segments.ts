import type { MachinesMacView, MachinesVM } from '@jterrazz/station-api';

import { macState, vmState } from '../words/shown';
import type { Shown } from '../words/shown';
import { machineRoute, OPENS_AN_ITEM } from './route';
import type { PlainPage, Route } from './route';
import { inSpace, SPACE_WORDS } from './spaces';
import { TITLES } from './titles';

// value: what a segment's popover lists: the spaces, the space's places, the whole fleet, or one Mac's VMs.
export type Siblings = 'fleet' | 'places' | 'spaces' | 'vms';

// value: one level of the path. `to` is where its name goes; a segment with no `siblings` is words, the page's own
// value: name or an item the engine names, and opens nothing.
export type Segment = {
    key: string;
    label: string;
    to?: Route | undefined;
    siblings?: Siblings | undefined;
    shown?: Shown | undefined;
    mac?: string | undefined;
    item?: undefined | { page: PlainPage; item: string };
};

// value: what the path reads of the open machine: its Mac's and its own name and standing.
export type Located = {
    mac: Pick<MachinesMacView, 'Name' | 'Standing'>;
    vm?: Pick<MachinesVM, 'Name' | 'Standing'> | undefined;
};

export type Path = { route: Route; found: Located | undefined; standing: Shown | undefined };

// value: the Machine space's pages a place segment lists, in the sidebar's order.
export const PLACES: readonly PlainPage[] = [
    'storage',
    'engine',
    'catalog-apps',
    'catalog-skills',
    'catalog-settings',
    'machine-files',
    'passwords',
];

function spaceOf(standing: Shown | undefined): Segment {
    return { key: 'space', label: SPACE_WORDS.machine.name, shown: standing, siblings: 'spaces' };
}

function machineSegments(
    route: Extract<Route, { page: 'machine' }>,
    found: Located | undefined,
): Segment[] {
    if (found === undefined) {
        return [{ key: 'machine', label: route.machine, siblings: 'fleet' }];
    }
    const mac: Segment = {
        key: 'mac',
        label: found.mac.Name,
        shown: macState(found.mac),
        siblings: 'fleet',
        to: machineRoute(found.mac.Name),
    };
    if (found.vm === undefined) {
        return [mac];
    }
    const vm: Segment = {
        key: 'vm',
        label: found.vm.Name,
        mac: found.mac.Name,
        shown: vmState(found.vm),
        siblings: 'vms',
        to: machineRoute(found.vm.Name),
    };
    return [mac, vm];
}

function placeSegments(route: Exclude<Route, { page: 'machine' }>): Segment[] {
    const { page, item } = route;
    const place: Segment = { key: 'place', label: TITLES[page], siblings: 'places', to: { page } };
    if (item === undefined || !OPENS_AN_ITEM.has(page)) {
        return [place];
    }
    return [place, { item: { item, page }, key: 'item', label: item }];
}

// invariant: a page of the Machine space opens on the space's segment; Home, Inbox, Settings and the gallery are
// invariant: the window's own, named alone; the frame with no engine is Home alone, whatever the route.
export function segmentsOf({ route, found, standing }: Path, quiet: boolean): Segment[] {
    if (quiet) {
        return [{ key: 'page', label: TITLES.home }];
    }
    if (!inSpace(route)) {
        return route.page === 'machine' ? [] : [{ key: 'page', label: TITLES[route.page] }];
    }
    const below = route.page === 'machine' ? machineSegments(route, found) : placeSegments(route);
    return [spaceOf(standing), ...below];
}
