import { machineRoute, plainPageOf, plainRoute, tabOf } from './route';
import type { Route } from './route';

export type Memory = Pick<Storage, 'getItem' | 'setItem'>;

const KEY = 'station.window.route';

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null;
}

function parsed(text: string): unknown {
    try {
        return JSON.parse(text) as unknown;
    } catch {
        return undefined;
    }
}

function routeOf(value: unknown): Route | undefined {
    if (!isRecord(value)) {
        return undefined;
    }
    const { page, machine, tab, file, item } = value;
    if (page === 'machine') {
        const known = tabOf(tab);
        return typeof machine === 'string' && machine !== '' && known
            ? machineRoute(machine, known)
            : undefined;
    }
    const plain = plainPageOf(page);
    if (plain === undefined) {
        return undefined;
    }
    return plainRoute(plain, file, item);
}

export function recalled(memory: Memory | undefined, withGallery: boolean): Route | undefined {
    const text = memory?.getItem(KEY) ?? null;
    const route = text === null ? undefined : routeOf(parsed(text));
    if (route?.page === 'gallery' && !withGallery) {
        return undefined;
    }
    return route;
}

export function remember(memory: Memory | undefined, route: Route): void {
    try {
        memory?.setItem(KEY, JSON.stringify(route));
    } catch {
        // hazard: a webview that refuses storage still runs; the next launch opens on Home.
    }
}
