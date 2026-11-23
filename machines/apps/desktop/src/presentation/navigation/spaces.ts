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
    return space === 'machine';
}

// value: where a page sits: the window's own places beside the space's sidebar, the global pages of the rail's foot
// value: alone in the content, every other page the Machine space's.
const REACH: Readonly<Record<PlainPage, 'global' | 'space' | 'window'>> = {
    'catalog-apps': 'space',
    'catalog-settings': 'space',
    'catalog-skills': 'space',
    engine: 'window',
    gallery: 'window',
    home: 'window',
    inbox: 'window',
    'machine-files': 'space',
    passwords: 'global',
    settings: 'global',
    storage: 'global',
};

export function inSpace(route: Route): boolean {
    return route.page === 'machine' || REACH[route.page] === 'space';
}

export function isGlobal(route: Route): boolean {
    return route.page !== 'machine' && REACH[route.page] === 'global';
}

// value: where the Machine space opens: the last of its pages this window showed, else the first Mac listed, else
// value: Machine files.
export function landingOf(last: Route | undefined, firstMac: string | undefined): Route {
    if (last !== undefined) {
        return last;
    }
    return firstMac === undefined ? { page: 'machine-files' } : machineRoute(firstMac);
}
