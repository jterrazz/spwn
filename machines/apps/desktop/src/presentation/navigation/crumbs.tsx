import type { ComponentType } from 'react';

import type { Found } from '../../domain/machines/machines';
import { machineRoute, OPENS_AN_ITEM } from './route';
import type { PlainPage, Route } from './route';
import type { Crumb } from './toolbar';

export const TITLES: Record<PlainPage, string> = {
    home: 'Home',
    inbox: 'Inbox',
    storage: 'Storage',
    'machine-files': 'Machine files',
    passwords: 'Passwords',
    settings: 'Settings',
    'catalog-apps': 'Apps',
    'catalog-skills': 'Skills',
    'catalog-settings': 'Machine settings',
    engine: 'Engine',
    gallery: 'Gallery',
};

type Crumbed = {
    route: Route;
    found: Found | undefined;
    go: (to: Route) => void;
    ItemName: ComponentType<{ page: PlainPage; item: string }>;
};

// value: the pages the sidebar groups under Catalog, which is a heading and no page of its own.
const IN_CATALOG: ReadonlySet<PlainPage> = new Set<PlainPage>([
    'catalog-apps',
    'catalog-settings',
    'catalog-skills',
    'machine-files',
]);

const CATALOG: Crumb = { label: 'Catalog' };

type PageRoute = Exclude<Route, { page: 'machine' }>;

function pageCrumbs(route: PageRoute, { go, ItemName }: Pick<Crumbed, 'go' | 'ItemName'>): Crumb[] {
    const { page, item } = route;
    const above = IN_CATALOG.has(page) ? [CATALOG] : [];
    if (item === undefined || !OPENS_AN_ITEM.has(page)) {
        return [...above, { label: TITLES[page] }];
    }
    const shelf = {
        label: TITLES[page],
        open: () => {
            go({ page });
        },
    };
    return [...above, shelf, { label: item, shown: <ItemName item={item} page={page} /> }];
}

export function crumbsOf({ route, found, go, ItemName }: Crumbed): Crumb[] {
    if (route.page !== 'machine') {
        return pageCrumbs(route, { go, ItemName });
    }
    if (found?.vm) {
        const mac = found.mac.Name;
        return [
            {
                label: mac,
                open: () => {
                    go(machineRoute(mac));
                },
            },
            { label: found.vm.Name },
        ];
    }
    return found ? [{ label: found.mac.Name }] : [{ label: route.machine }];
}
