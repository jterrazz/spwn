import { describe, expect, test } from 'vitest';

import { macState } from '../../words/shown';
import { machineRoute } from '../route';
import type { Route } from '../route';
import { fleetRoute, SECTIONS, segmentsOf } from '../segments';
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

        // Then - Fleet opens the card, the Mac the fleet, the VM its Mac's VMs
        expect(path).toStrictEqual([
            { label: 'Fleet', siblings: 'card', to: undefined },
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

    test('a page of the space starts at its section, an item at its shelf then its name', () => {
        // Given - Storage, then an app on its shelf
        const storage = pathOf({ page: 'storage' });
        const item = pathOf({ item: 'gh', page: 'catalog-apps' });

        // Then - the section opens the card, the item is words
        expect(storage).toStrictEqual([
            { label: 'Storage', siblings: 'card', to: { page: 'storage' } },
        ]);
        expect(item.map((segment) => segment.label)).toStrictEqual(['Apps', 'gh']);
        expect(item[1]?.siblings).toBeUndefined();
    });

    test("home, Inbox and Settings are the window's own, named alone, and open the card too", () => {
        // Given - each of the window's own pages
        const named = (['home', 'inbox', 'settings'] as const).map((page) => pathOf({ page }));

        // Then - one segment, its page's name, opening the card
        expect(named).toStrictEqual([
            [{ label: 'Home', siblings: 'card', to: undefined }],
            [{ label: 'Inbox', siblings: 'card', to: undefined }],
            [{ label: 'Settings', siblings: 'card', to: undefined }],
        ]);
    });

    test('the frame with no engine is Home alone, as words, whatever the route', () => {
        // Given - a route to linux-1 while no engine answers
        const path = pathOf(machineRoute('linux-1'), LINUX_1, true);

        // Then - Home, and no segment that would open the card
        expect(path).toStrictEqual([{ label: 'Home', siblings: undefined, to: undefined }]);
    });

    test('the Machine space lists Home, the fleet, its shelves and places; a space not built lists none', () => {
        // Given - the card's sections of each space
        const sections = SECTIONS;

        // Then - in their order
        expect(sections).toStrictEqual({
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
        });
    });

    test("the card's Fleet goes to the last machine shown, else the first Mac, else nowhere", () => {
        // Given - a window that showed linux-1's Ports, one that showed only Storage, one with no Mac
        const shown = fleetRoute(machineRoute('linux-1', 'Ports'), 'macbook-pro');
        const section = fleetRoute({ page: 'storage' }, 'macbook-pro');
        const none = fleetRoute(undefined, undefined);

        // Then - linux-1 where it was left, the first Mac, nothing
        expect([shown, section, none]).toStrictEqual([
            machineRoute('linux-1', 'Ports'),
            machineRoute('macbook-pro'),
            undefined,
        ]);
    });
});
