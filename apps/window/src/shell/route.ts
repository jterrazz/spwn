export const TABS = ['Overview', 'Activity', 'Packages', 'Ports', 'Accounts', 'Storage'] as const;

export type Tab = (typeof TABS)[number];

const FOLDED: ReadonlyMap<string, Tab> = new Map([['Logs', 'Activity']]);

export function tabOf(value: unknown): Tab | undefined {
    if (typeof value !== 'string') {
        return undefined;
    }
    return TABS.find((known) => known === value) ?? FOLDED.get(value);
}

export const TAB_PANEL = 'machine-tab-panel';

export const PLAIN_PAGES = [
    'catalog-apps',
    'catalog-settings',
    'catalog-skills',
    'gallery',
    'home',
    'inbox',
    'machine-files',
    'storage',
] as const;

export type PlainPage = (typeof PLAIN_PAGES)[number];

export const IN_CATALOG: ReadonlySet<PlainPage> = new Set<PlainPage>([
    'catalog-apps',
    'catalog-settings',
    'catalog-skills',
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

// hazard: a route an earlier window stored or linked may name the page `catalog`; it opens the first shelf, Apps.
export function plainPageOf(value: unknown): PlainPage | undefined {
    return value === 'catalog' ? 'catalog-apps' : oneOf(PLAIN_PAGES, value);
}

function named(value: unknown): string | undefined {
    return typeof value === 'string' && value !== '' ? value : undefined;
}

export function plainRoute(page: PlainPage, file: unknown, item: unknown): Route {
    const route: PlainRoute = { page };
    const opened = named(file);
    const chosen = IN_CATALOG.has(page) ? named(item) : undefined;
    if (opened !== undefined) {
        route.file = opened;
    }
    if (chosen !== undefined) {
        route.item = chosen;
    }
    return route;
}
