import type { OverviewNeed } from '@jterrazz/station-api';
import { describe, expect, test } from 'vitest';

import { everything, matching } from '../finds';
import type { Group, Held } from '../finds';
import { machineRoute } from '../route';
import type { PlainPage } from '../route';

const TITLES: Record<PlainPage, string> = {
    'catalog-apps': 'Apps',
    'catalog-settings': 'Settings',
    'catalog-skills': 'Skills',
    gallery: 'Gallery',
    home: 'Home',
    inbox: 'Inbox',
    'machine-files': 'Machine files',
    storage: 'Storage',
};

const NEED: OverviewNeed = {
    Args: null,
    Broken: true,
    Call: '',
    Command: '',
    Machine: 'linux-1',
    Path: '',
    Section: 'ports',
    Severity: 'broken',
    Subject: 'web',
    Text: 'web does not answer on :3000',
};

const HELD: Held = {
    apps: [
        { category: 'editors', name: 'zed' },
        { category: 'runtimes', name: 'node' },
    ],
    macs: [
        { Name: 'mini', VMs: null },
        {
            Name: 'book',
            VMs: [
                { Lifetime: 'ephemeral', Name: 'linux-1' },
                { Lifetime: 'always-on', Name: 'api' },
            ],
        },
    ],
    needs: [NEED],
    routeOfNeed: (need) => machineRoute(need.Machine, 'Ports'),
    settings: [{ category: 'dock', name: 'dock-autohide' }],
    titles: TITLES,
    withGallery: false,
};

function shown(groups: readonly Group[]) {
    return groups.map((group) => [group.kind, group.results.map((result) => result.label)]);
}

describe('finds', () => {
    test("an empty query lists what the sidebar holds, in the sidebar's order, machines as listed", () => {
        // Given - two Macs, two VMs, a need and three packages held
        const groups = matching(everything(HELD), '');

        // Then - places, the machines each Mac then its VMs, Machine files, then the catalog's shelves, no action; tabs, needs and packages wait for a query
        expect(shown(groups)).toStrictEqual([
            ['Places', ['Home', 'Inbox', 'Storage']],
            ['Machines', ['mini', 'book', 'linux-1', 'api']],
            ['Machine files', ['Machine files']],
            ['Catalog', ['Apps', 'Skills', 'Settings']],
        ]);
    });

    test('a place is a page, Machine files is setup, and a shelf says it is the catalog', () => {
        // Given - the empty query's groups
        const groups = matching(everything(HELD), '');

        // Then - each row carries the trail of its section
        const details = (kind: string) =>
            groups.find((group) => group.kind === kind)?.results.map((result) => result.detail);
        expect(details('Places')).toStrictEqual(['page', 'page', 'page']);
        expect(details('Machine files')).toStrictEqual(['setup']);
        expect(details('Catalog')).toStrictEqual(['catalog', 'catalog', 'catalog']);
    });

    test('the word catalog finds its three shelves, each opening its own page', () => {
        // Given - a query for the catalog
        const groups = matching(everything(HELD), 'catalog');

        // Then - Apps, Skills and Settings, in the sidebar's order
        expect(shown(groups)).toStrictEqual([['Catalog', ['Apps', 'Skills', 'Settings']]]);
        expect(groups[0]?.results.map((result) => result.to)).toStrictEqual([
            { route: { page: 'catalog-apps' } },
            { route: { page: 'catalog-skills' } },
            { route: { page: 'catalog-settings' } },
        ]);
    });

    test('the Add a Mac action is found, shown only to a query', () => {
        // Given - every result Find holds

        // When - it is asked with no query, then for add a mac
        const unasked = matching(everything(HELD), '');
        const asked = matching(everything(HELD), 'add a mac');

        // Then - the action waits for a query, and acts instead of opening a page
        expect(shown(unasked).map(([kind]) => kind)).not.toContain('Actions');
        expect(shown(asked)).toStrictEqual([['Actions', ['Add a Mac']]]);
        expect(asked[0]?.results[0]?.to).toStrictEqual({ act: 'add-mac' });
    });

    test('this Mac is named so, any other Mac is a Mac', () => {
        // Given - book as the Mac the window runs on
        const groups = matching(
            everything({
                ...HELD,
                macs: HELD.macs.map((mac) => ({ ...mac, ThisMac: mac.Name === 'book' })),
            }),
            'book',
        );

        // Then - book's row says This Mac
        expect(groups[0]?.results[0]?.detail).toBe('This Mac');
    });

    test('every word of a query is found in a name or its kind, in any case', () => {
        // Given - a query naming a machine and a tab
        const groups = matching(everything(HELD), 'LINUX activity');

        // Then - the one tab of linux-1 it names, and no other result
        expect(shown(groups)).toStrictEqual([['Tabs', ['linux-1 › Activity']]]);
        expect(groups[0]?.results[0]?.to).toStrictEqual({
            route: machineRoute('linux-1', 'Activity'),
        });
    });

    test("a machine's tabs come in the tab bar's order, machine by machine, and no row repeats its group", () => {
        // Given - a query every tab of linux-1 and of api answers
        const groups = matching(everything(HELD), '›');

        // Then - api's six tabs, then linux-1's, each in the order of the tab bar, with no detail of their own
        const tabs = groups.find((group) => group.kind === 'Tabs')?.results ?? [];
        expect(tabs.slice(0, 6).map((result) => result.label)).toStrictEqual([
            'api › Overview',
            'api › Activity',
            'api › Packages',
            'api › Ports',
            'api › Accounts',
            'api › Storage',
        ]);
        expect(tabs[6]?.label).toBe('book › Overview');
        expect(tabs.every((result) => result.detail === '')).toBe(true);
    });

    test("a package is found by its name and opens its page, a need opens where the need's row does", () => {
        // Given - queries for a setting, an app's category, and a need's words
        const setting = matching(everything(HELD), 'dock');
        const runtime = matching(everything(HELD), 'runtimes');
        const need = matching(everything(HELD), 'answer');

        // Then - each lands where its page shows it
        expect(setting[0]?.results[0]?.to).toStrictEqual({
            route: { item: 'dock-autohide', page: 'catalog-settings' },
        });
        expect(shown(runtime)).toStrictEqual([['Packages', ['node']]]);
        expect(need[0]?.results[0]?.to).toStrictEqual({ route: machineRoute('linux-1', 'Ports') });
    });

    test('the gallery is found only in a window that holds it, and only by its name', () => {
        // Given - the same query in a production window, then a proof window, then the proof window's empty query
        const production = matching(everything(HELD), 'gallery');
        const proof = matching(everything({ ...HELD, withGallery: true }), 'gallery');
        const unasked = matching(everything({ ...HELD, withGallery: true }), '');

        // Then - only the proof window finds it, and its empty query does not list it
        expect(production).toStrictEqual([]);
        expect(shown(proof)).toStrictEqual([['Places', ['Gallery']]]);
        expect(shown(unasked)).toContainEqual(['Places', ['Home', 'Inbox', 'Storage']]);
    });

    test('a query nothing holds finds nothing', () => {
        // Given - a word no name carries
        // Then - no group at all
        expect(matching(everything(HELD), 'kubernetes')).toStrictEqual([]);
    });
});
