import { machineRoute } from './route';
import type { PlainPage, Route } from './route';

export const SPACES = ['machine', 'agent', 'os'] as const;

export type Space = (typeof SPACES)[number];

// value: the three spaces of Spwn: Machine is the one built, Agent and OS are named and wait.
export const SPACE_WORDS: Readonly<Record<Space, { name: string; line: string }>> = {
    agent: { line: 'The workforce: personalities, roles, focus', name: 'Agent' },
    machine: { line: 'The fleet, access, security, documents', name: 'Machine' },
    os: { line: 'Flows, projects, vision', name: 'OS' },
};

export const SOON_WORD = 'Soon';

export function opensNow(space: Space): boolean {
    return space !== 'agent';
}

// value: where a page sits: the window's own places beside the space's sidebar, the global pages of the rail's foot
// value: alone in the content, the wiki the OS space's, alone too as the OS has no sidebar yet, every other page the
// value: Machine space's, Storage and Engine among them, which show every Mac at once.
const REACH: Readonly<Record<PlainPage, 'global' | 'os' | 'space' | 'window'>> = {
    'catalog-apps': 'space',
    'catalog-settings': 'space',
    'catalog-skills': 'space',
    engine: 'space',
    gallery: 'window',
    home: 'window',
    inbox: 'window',
    'machine-files': 'space',
    passwords: 'global',
    settings: 'global',
    storage: 'space',
    wiki: 'os',
};

export function inSpace(route: Route): boolean {
    return route.page === 'machine' || REACH[route.page] === 'space';
}

export function spaceOf(route: Route): Space | undefined {
    if (inSpace(route)) {
        return 'machine';
    }
    return route.page !== 'machine' && REACH[route.page] === 'os' ? 'os' : undefined;
}

export function standsAlone(route: Route): boolean {
    return (
        route.page !== 'machine' && (REACH[route.page] === 'global' || REACH[route.page] === 'os')
    );
}

// value: where the Machine space opens: the last of its pages this window showed, else the first Mac listed, else
// value: Machine files.
export function landingOf(last: Route | undefined, firstMac: string | undefined): Route {
    if (last !== undefined) {
        return last;
    }
    return firstMac === undefined ? { page: 'machine-files' } : machineRoute(firstMac);
}
