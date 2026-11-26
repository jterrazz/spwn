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

// value: where a page sits: the window's own places beside the space's column, Settings alone in the content, every
// value: other page the Machine space's, Storage, Engine and Passwords among them.
const REACH: Readonly<Record<PlainPage, 'global' | 'space' | 'window'>> = {
    'catalog-apps': 'space',
    'catalog-settings': 'space',
    'catalog-skills': 'space',
    engine: 'space',
    gallery: 'window',
    home: 'window',
    inbox: 'window',
    'machine-files': 'space',
    passwords: 'space',
    settings: 'global',
    storage: 'space',
};

export function inSpace(route: Route): boolean {
    return route.page === 'machine' || REACH[route.page] === 'space';
}

export function isGlobal(route: Route): boolean {
    return route.page !== 'machine' && REACH[route.page] === 'global';
}

// invariant: read from the route, never stored, so Back and Forward move the underline; Settings is in no space.
export function spaceOfRoute(route: Route): Space | undefined {
    return isGlobal(route) ? undefined : 'machine';
}

// value: where the Machine space opens: the last of its pages this window showed, else the first Mac listed, else
// value: Machine files.
export function landingOf(last: Route | undefined, firstMac: string | undefined): Route {
    if (last !== undefined) {
        return last;
    }
    return firstMac === undefined ? { page: 'machine-files' } : machineRoute(firstMac);
}
