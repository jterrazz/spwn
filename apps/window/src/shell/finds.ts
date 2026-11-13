import type { MachinesVM, OverviewNeed } from '@jterrazz/station-api';

import { machineKindOf } from '../engine/provisional';
import type { MachineKind } from '../engine/provisional';
import { vmKindWord } from '../engine/shown';
import { THIS_MAC_WORD } from '../engine/words';
import { machineRoute, TABS } from './route';
import type { PlainPage, Route } from './route';

export const KINDS = [
    'Places',
    'Machines',
    'Tabs',
    'Library',
    'Needs',
    'Packages',
    'Actions',
] as const;

export type Kind = (typeof KINDS)[number];

export type Action = 'new-machine' | 'sidebar';

export type Result = {
    kind: Kind;
    label: string;
    detail: string;
    to: { action: Action } | { route: Route };
    mark?: undefined | { name: string; kind: string };
    machine?: undefined | { name: string; kind: 'mac' | MachineKind };
};

export type Group = { kind: Kind; results: readonly Result[] };

export type Held = {
    titles: Readonly<Record<PlainPage, string>>;
    withGallery: boolean;
    macs: readonly {
        Name: string;
        ThisMac?: boolean;
        VMs: null | readonly { Name: string; Lifetime: MachinesVM['Lifetime'] }[];
    }[];
    needs: readonly OverviewNeed[];
    routeOfNeed: (need: OverviewNeed) => Route;
    apps: readonly { name: string; category: string; skill?: boolean }[];
    settings: readonly { name: string; category: string }[];
};

const PLACES: readonly PlainPage[] = ['home', 'inbox', 'storage'];
const LIBRARY: readonly PlainPage[] = [
    'machine-files',
    'catalog',
    'catalog-apps',
    'catalog-skills',
    'catalog-settings',
];

const WITH_A_QUERY: ReadonlySet<Kind> = new Set<Kind>(['Packages', 'Tabs']);
const PAGE_DETAILS: Partial<Record<Kind, string>> = { Library: 'setup', Places: 'page' };

function page(held: Held, kind: Kind, one: PlainPage): Result {
    return {
        detail: PAGE_DETAILS[kind] ?? '',
        kind,
        label: held.titles[one],
        to: { route: { page: one } },
    };
}

function machinesOf(held: Held): readonly Result[] {
    return held.macs.flatMap((mac): Result[] => [
        {
            detail: mac.ThisMac === true ? THIS_MAC_WORD : 'Mac',
            kind: 'Machines',
            label: mac.Name,
            machine: { kind: 'mac', name: mac.Name },
            to: { route: machineRoute(mac.Name) },
        },
        ...(mac.VMs ?? []).map((vm): Result => ({
            detail: `${vmKindWord(vm)} on ${mac.Name}`,
            kind: 'Machines',
            label: vm.Name,
            machine: { kind: machineKindOf(vm), name: vm.Name },
            to: { route: machineRoute(vm.Name) },
        })),
    ]);
}

function tabsOf(held: Held): readonly Result[] {
    const names = held.macs.flatMap((mac) => [mac.Name, ...(mac.VMs ?? []).map((vm) => vm.Name)]);
    return names.flatMap((name) =>
        TABS.map((tab): Result => ({
            detail: '',
            kind: 'Tabs',
            label: `${name} › ${tab}`,
            to: { route: machineRoute(name, tab) },
        })),
    );
}

function shelfOf(shelf: 'apps' | 'settings', skill: boolean | undefined): PlainPage {
    if (shelf === 'settings') {
        return 'catalog-settings';
    }
    return skill === true ? 'catalog-skills' : 'catalog-apps';
}

function packageOf(
    shelf: 'apps' | 'settings',
    item: { name: string; category: string; skill?: boolean },
): Result {
    return {
        detail: sentenceOf([shelf === 'apps' ? 'app' : 'setting', item.category]),
        kind: 'Packages',
        label: item.name,
        mark: { kind: shelf === 'apps' ? 'app' : 'setting', name: item.name },
        to: { route: { item: item.name, page: shelfOf(shelf, item.skill) } },
    };
}

function packagesOf(held: Held): readonly Result[] {
    return [
        ...held.apps.map((item) => packageOf('apps', item)),
        ...held.settings.map((item) => packageOf('settings', item)),
    ];
}

function sentenceOf(parts: readonly string[]): string {
    return parts.filter((part) => part !== '').join(' · ');
}

export function everything(held: Held): readonly Result[] {
    const library = held.withGallery ? [...LIBRARY, 'gallery' as const] : LIBRARY;
    return [
        ...PLACES.map((one) => page(held, 'Places', one)),
        ...held.needs.map((need): Result => ({
            detail: need.Machine,
            kind: 'Needs',
            label: need.Text,
            to: { route: held.routeOfNeed(need) },
        })),
        ...machinesOf(held),
        ...tabsOf(held),
        ...library.map((one) => page(held, 'Library', one)),
        ...packagesOf(held),
        { detail: '', kind: 'Actions', label: 'New machine', to: { action: 'new-machine' } },
        {
            detail: '',
            kind: 'Actions',
            label: 'Show or hide the sidebar',
            to: { action: 'sidebar' },
        },
    ];
}

function holds(result: Result, terms: readonly string[]): boolean {
    const text = `${result.label} ${result.detail}`.toLowerCase();
    return terms.every((term) => text.includes(term));
}

export function resultKey(result: Result): string {
    return `${result.kind}\u0000${result.label}\u0000${result.detail}`;
}

function machineOf(result: Result): string {
    return 'route' in result.to && result.to.route.page === 'machine'
        ? result.to.route.machine
        : '';
}

// invariant: a machine's tabs keep the tab bar's order, the order everything() lists them in, so only their
// invariant: machines are sorted, and toSorted is stable. The machines keep the sidebar's order, each Mac then its
// invariant: VMs, as machines.list sends them, and the library rows keep the sidebar's.
function inOrder(kind: Kind): (a: Result, b: Result) => number {
    if (kind === 'Machines' || kind === 'Library') {
        return () => 0;
    }
    return kind === 'Tabs'
        ? (a, b) => machineOf(a).localeCompare(machineOf(b), 'en')
        : (a, b) => a.label.localeCompare(b.label, 'en');
}

export function matching(all: readonly Result[], query: string): readonly Group[] {
    const terms = query
        .trim()
        .toLowerCase()
        .split(/\s+/u)
        .filter((term) => term !== '');
    const seen = new Set<string>();
    const kept = all.filter((result) => {
        const key = resultKey(result);
        if (seen.has(key) || (terms.length === 0 && WITH_A_QUERY.has(result.kind))) {
            return false;
        }
        seen.add(key);
        return holds(result, terms);
    });
    return KINDS.map((kind) => ({
        kind,
        results: kept.filter((result) => result.kind === kind).toSorted(inOrder(kind)),
    })).filter((group) => group.results.length > 0);
}
