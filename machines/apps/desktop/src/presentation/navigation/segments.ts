import type { MachinesMacView, MachinesVM } from '@jterrazz/station-api';

import { macState, vmState } from '../words/shown';
import type { Shown } from '../words/shown';
import { machineRoute, OPENS_AN_ITEM } from './route';
import type { PlainPage, Route } from './route';
import { inSpace } from './spaces';
import type { Space } from './spaces';
import { TITLES } from './titles';

// value: what a segment's popover lists: the card (the space's sections beside the fleet), the whole fleet, or one
// value: Mac's VMs.
export type Siblings = 'card' | 'fleet' | 'vms';

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

export type Path = { route: Route; found: Located | undefined };

export type Section = 'fleet' | PlainPage;

// value: the sections the card lists for each space, in its order: Home first, then the fleet, the Library's
// value: shelves, Storage, Passwords and the engine; a space not built lists none.
export const SECTIONS: Readonly<Record<Space, readonly Section[]>> = {
    agent: [],
    machine: [
        'home',
        'fleet',
        'catalog-apps',
        'catalog-skills',
        'catalog-settings',
        'machine-files',
        'storage',
        'passwords',
        'engine',
    ],
    os: [],
};

// value: where the card's Fleet goes, as there is no page of the fleet: the last machine this window showed, else
// value: the first Mac listed, else nowhere.
export function fleetRoute(
    last: Route | undefined,
    firstMac: string | undefined,
): Route | undefined {
    if (last?.page === 'machine') {
        return last;
    }
    return firstMac === undefined ? undefined : machineRoute(firstMac);
}

// value: the section a machine's path starts at: the fleet, unmarked, which opens the card.
const FLEET: Segment = { key: 'fleet', label: 'Fleet', siblings: 'card' };

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
    const place: Segment = { key: 'place', label: TITLES[page], siblings: 'card', to: { page } };
    if (item === undefined || !OPENS_AN_ITEM.has(page)) {
        return [place];
    }
    return [place, { item: { item, page }, key: 'item', label: item }];
}

// invariant: the path starts at the section, never at the space, which is the bar's own switch: a machine opens on
// invariant: Fleet, a page of the Machine space on its place; Home, Inbox, Settings and the gallery are the window's
// invariant: own, named alone. The first segment opens the card on every page; the frame with no engine is Home
// invariant: alone, as words, whatever the route.
export function segmentsOf({ route, found }: Path, quiet: boolean): Segment[] {
    if (quiet) {
        return [{ key: 'page', label: TITLES.home }];
    }
    if (route.page === 'machine') {
        return [FLEET, ...machineSegments(route, found)];
    }
    return inSpace(route)
        ? placeSegments(route)
        : [{ key: 'page', label: TITLES[route.page], siblings: 'card' }];
}
