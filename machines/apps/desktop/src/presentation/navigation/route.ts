import type { MachinesSections } from '@jterrazz/station-api';

export const TABS = ['Overview', 'Activity', 'Packages', 'Ports', 'Sign-ins', 'Storage'] as const;

export type Tab = (typeof TABS)[number];

const SECTIONS_OF: Readonly<Record<Tab, (sections: MachinesSections) => boolean>> = {
    Activity: (sections) => sections.Activity,
    Overview: (sections) => sections.Overview,
    Packages: (sections) => sections.Apps || sections.Settings,
    Ports: (sections) => sections.Ports,
    'Sign-ins': (sections) => sections.Accounts,
    Storage: (sections) => sections.Disk,
};

// value: a Mac's disk is a section of the Storage page, so a Mac has every tab but Storage.
const MAC_TABS: readonly Tab[] = TABS.filter((tab) => tab !== 'Storage');

// invariant: a VM's tabs are the ones its engine says it has sections for, every tab when it sends none.
export function tabsOf(
    vm: undefined | { Sections?: MachinesSections | undefined },
): readonly Tab[] {
    if (vm === undefined) {
        return MAC_TABS;
    }
    const { Sections } = vm;
    return Sections === undefined ? TABS : TABS.filter((tab) => SECTIONS_OF[tab](Sections));
}

// value: where a route of a Mac's tab opens: its Storage, from a link or a relaunch, is the Storage page.
export function settledRoute(route: Route, isMac: boolean): Route {
    return route.page === 'machine' && isMac && !MAC_TABS.includes(route.tab)
        ? { page: 'storage' }
        : route;
}

export function tabOf(value: unknown): Tab | undefined {
    return TABS.find((known) => known === value);
}

export const TAB_PANEL = 'machine-tab-panel';

export const PLAIN_PAGES = [
    'catalog-apps',
    'catalog-settings',
    'catalog-skills',
    'engine',
    'gallery',
    'home',
    'inbox',
    'machine-files',
    'passwords',
    'settings',
    'storage',
    'wiki',
] as const;

export type PlainPage = (typeof PLAIN_PAGES)[number];

// value: the pages that open on one of their items, a catalog item, a vault entry or a wiki page, named in the
// value: route's `item`.
export const OPENS_AN_ITEM: ReadonlySet<PlainPage> = new Set<PlainPage>([
    'catalog-apps',
    'catalog-settings',
    'catalog-skills',
    'passwords',
    'wiki',
]);

type PlainRoute = { page: PlainPage; file?: string | undefined; item?: string | undefined };

export type Route = PlainRoute | { page: 'machine'; machine: string; tab: Tab };

export type History = { current: Route; back: readonly Route[]; forward: readonly Route[] };

const DEPTH = 40;

export function sameRoute(a: Route, b: Route): boolean {
    if (a.page === 'machine' && b.page === 'machine') {
        return a.machine === b.machine && a.tab === b.tab;
    }
    if (a.page === 'machine' || b.page === 'machine') {
        return false;
    }
    return (
        a.page === b.page && (a.file ?? '') === (b.file ?? '') && (a.item ?? '') === (b.item ?? '')
    );
}

export function historyAt(current: Route): History {
    return { back: [], current, forward: [] };
}

export function navigate(history: History, to: Route): History {
    if (sameRoute(history.current, to)) {
        return history;
    }
    return { back: [...history.back, history.current].slice(-DEPTH), current: to, forward: [] };
}

export function replace(history: History, to: Route): History {
    return sameRoute(history.current, to) ? history : { ...history, current: to };
}

export function goBack(history: History): History {
    const previous = history.back.at(-1);
    if (!previous) {
        return history;
    }
    return {
        back: history.back.slice(0, -1),
        current: previous,
        forward: [history.current, ...history.forward].slice(0, DEPTH),
    };
}

export function goForward(history: History): History {
    const [next, ...rest] = history.forward;
    if (!next) {
        return history;
    }
    return { back: [...history.back, history.current].slice(-DEPTH), current: next, forward: rest };
}

export function machineRoute(machine: string, tab: Tab = 'Overview'): Route {
    return { page: 'machine', machine, tab };
}

function oneOf<T extends string>(values: readonly T[], value: unknown): T | undefined {
    return values.find((known) => known === value);
}

export function startRoute(query: URLSearchParams): Route {
    const machine = query.get('machine');
    if (machine !== null && machine !== '') {
        return machineRoute(machine, tabOf(query.get('tab')) ?? 'Overview');
    }
    const page = plainPageOf(query.get('page')) ?? 'home';
    return plainRoute(page, query.get('file'), query.get('item'));
}

export function plainPageOf(value: unknown): PlainPage | undefined {
    return oneOf(PLAIN_PAGES, value);
}

function named(value: unknown): string | undefined {
    return typeof value === 'string' && value !== '' ? value : undefined;
}

export function plainRoute(page: PlainPage, file: unknown, item: unknown): Route {
    const route: PlainRoute = { page };
    const opened = named(file);
    const chosen = OPENS_AN_ITEM.has(page) ? named(item) : undefined;
    if (opened !== undefined) {
        route.file = opened;
    }
    if (chosen !== undefined) {
        route.item = chosen;
    }
    return route;
}
