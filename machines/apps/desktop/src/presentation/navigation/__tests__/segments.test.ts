import { describe, expect, test } from 'vitest';

import { macState } from '../../words/shown';
import { machineRoute } from '../route';
import type { Route } from '../route';
import { segmentsOf } from '../segments';
import type { Located } from '../segments';

const LINUX_1: Located = {
    mac: { Name: 'macbook-pro', Standing: 'online' },
    vm: { Name: 'linux-1', Standing: 'running' },
};

function pathOf(route: Route, found?: Located, quiet = false) {
    return segmentsOf({ found, route }, quiet).map((segment) => ({
        label: segment.label,
        siblings: segment.siblings,
        to: segment.to,
    }));
}

describe('segments', () => {
    test("a VM's path is Fleet, its Mac, then itself, each opening its siblings", () => {
        // Given - linux-1 on macbook-pro
        const path = pathOf(machineRoute('linux-1'), LINUX_1);

        // Then - Fleet lists the places, the Mac the fleet, the VM its Mac's VMs
        expect(path).toStrictEqual([
            { label: 'Fleet', siblings: 'places', to: undefined },
            { label: 'macbook-pro', siblings: 'fleet', to: machineRoute('macbook-pro') },
            { label: 'linux-1', siblings: 'vms', to: machineRoute('linux-1') },
        ]);
    });

    test('the Fleet segment carries no mark, and a Mac its own standing', () => {
        // Given - macbook-pro opened
        const [fleet, mac] = segmentsOf(
            { found: { mac: LINUX_1.mac }, route: machineRoute('macbook-pro') },
            false,
        );

        // Then - no roll-up on Fleet, the Mac's word as the engine says it
        expect(fleet?.shown).toBeUndefined();
        expect(mac?.shown).toStrictEqual(macState({ Standing: 'online' }));
    });

    test('a page of the space starts at its place, an item at its shelf then its name', () => {
        // Given - Storage, then an app on its shelf
        const storage = pathOf({ page: 'storage' });
        const item = pathOf({ item: 'gh', page: 'catalog-apps' });

        // Then - the place lists its siblings, the item is words
        expect(storage).toStrictEqual([
            { label: 'Storage', siblings: 'places', to: { page: 'storage' } },
        ]);
        expect(item.map((segment) => segment.label)).toStrictEqual(['Apps', 'gh']);
        expect(item[1]?.siblings).toBeUndefined();
    });

    test("home, Inbox and Settings are the window's own, named alone", () => {
        // Given - each of the window's own pages
        const named = (['home', 'inbox', 'settings'] as const).map((page) => pathOf({ page }));

        // Then - one segment, its page's name, opening nothing
        expect(named).toStrictEqual([
            [{ label: 'Home', siblings: undefined, to: undefined }],
            [{ label: 'Inbox', siblings: undefined, to: undefined }],
            [{ label: 'Settings', siblings: undefined, to: undefined }],
        ]);
    });

    test('passwords is a place of the Machine space', () => {
        // Given - the vault's list
        const path = pathOf({ page: 'passwords' });

        // Then - its place alone, which lists the others
        expect(path).toStrictEqual([
            { label: 'Passwords', siblings: 'places', to: { page: 'passwords' } },
        ]);
    });

    test('the frame with no engine is Home alone, whatever the route', () => {
        // Given - a route to linux-1 while no engine answers
        const path = pathOf(machineRoute('linux-1'), LINUX_1, true);

        // Then - Home, and no segment that would open the fleet
        expect(path).toStrictEqual([{ label: 'Home', siblings: undefined, to: undefined }]);
    });
});
