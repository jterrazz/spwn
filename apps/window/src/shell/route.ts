export const TABS = [
    'Overview',
    'Activity',
    'Packages',
    'Ports',
    'Accounts',
    'Storage',
    'Logs',
] as const;

export type Tab = (typeof TABS)[number];

export const TAB_PANEL = 'machine-tab-panel';

export const PLAIN_PAGES = [
    'catalog',
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

export type Route =
    | { page: 'machine'; machine: string; tab: Tab }
    | { page: PlainPage; file?: string | undefined };

export type History = { current: Route; back: readonly Route[]; forward: readonly Route[] };

const DEPTH = 40;

export function sameRoute(a: Route, b: Route): boolean {
    if (a.page === 'machine' && b.page === 'machine') {
        return a.machine === b.machine && a.tab === b.tab;
    }
    if (a.page === 'machine' || b.page === 'machine') {
        return false;
    }
    return a.page === b.page && (a.file ?? '') === (b.file ?? '');
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
        return machineRoute(machine, oneOf(TABS, query.get('tab')) ?? 'Overview');
    }
    const page = oneOf(PLAIN_PAGES, query.get('page')) ?? 'home';
    const file = query.get('file');
    return file !== null && file !== '' ? { file, page } : { page };
}
