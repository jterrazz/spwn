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

// value: the pages of the window itself, reached from the rail whatever the space: every other page is Machine's.
const THE_WINDOWS: ReadonlySet<PlainPage> = new Set<PlainPage>([
    'engine',
    'gallery',
    'home',
    'inbox',
    'settings',
]);

export function inSpace(route: Route): boolean {
    return route.page === 'machine' || !THE_WINDOWS.has(route.page);
}

// value: where the Machine space opens: the last of its pages this window showed, else the first Mac listed, else
// value: Machine files.
export function landingOf(last: Route | undefined, firstMac: string | undefined): Route {
    if (last !== undefined) {
        return last;
    }
    return firstMac === undefined ? { page: 'machine-files' } : machineRoute(firstMac);
}
