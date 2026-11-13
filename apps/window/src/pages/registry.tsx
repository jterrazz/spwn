import { Suspense } from 'react';

import { TITLES } from '../shell/window';
import type { PageProps, Pages } from '../shell/window';
import { CatalogPage } from './catalog/catalog';
import { ShelfPage } from './catalog/shelf';
import { HomePage } from './home/home';
import { InboxPage } from './inbox/inbox';
import { GestureConfirm } from './kit/gestures';
import { MachineFilesPage } from './machine-files/machine-files';
import { MachineHeader } from './machine/header';
import { MachineTab } from './machine/machine';
import { routeOfNeed } from './needs/needs';
import { NewMachine } from './new-machine/new-machine';
import { PlaceholderPage } from './placeholder';
import { StoragePage } from './storage/storage';

function Head({ route, found, go, newMachine }: PageProps) {
    return route.page === 'machine' && found ? (
        <MachineHeader found={found} go={go} onNewMachine={newMachine} tab={route.tab} />
    ) : null;
}

function Body({ route, found, home, go, Gallery, newMachine }: PageProps) {
    switch (route.page) {
        case 'home': {
            return <HomePage go={go} onNewMachine={newMachine} {...home} />;
        }
        case 'inbox': {
            return <InboxPage go={go} {...home} />;
        }
        case 'storage': {
            return <StoragePage go={go} macs={home.macs} />;
        }
        case 'machine': {
            return found ? (
                <MachineTab found={found} go={go} tab={route.tab} />
            ) : (
                <PlaceholderPage title={route.machine} />
            );
        }
        case 'machine-files': {
            return (
                <MachineFilesPage
                    key={route.file ?? ''}
                    onNewMachine={newMachine}
                    opened={route.file}
                />
            );
        }
        case 'catalog': {
            return <CatalogPage go={go} />;
        }
        case 'catalog-apps': {
            return <ShelfPage shelf="apps" />;
        }
        case 'catalog-skills': {
            return <ShelfPage shelf="skills" />;
        }
        case 'catalog-settings': {
            return <ShelfPage shelf="settings" />;
        }
        case 'gallery': {
            return Gallery ? (
                <Suspense fallback={null}>
                    <Gallery />
                </Suspense>
            ) : (
                <PlaceholderPage title={TITLES.gallery} />
            );
        }
    }
}

export const PAGES: Pages = { Body, Confirm: GestureConfirm, Head, NewMachine, routeOfNeed };
