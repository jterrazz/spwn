import { Suspense } from 'react';

import { routeOfNeed } from '../features/overview/needs';
import { GestureConfirm } from '../kit/gestures';
import { TITLES } from '../navigation/crumbs';
import type { PlainPage } from '../navigation/route';
import type { PageProps, Pages } from '../navigation/window';
import { AddMac } from './add-mac/add-mac';
import { ItemPage } from './catalog/item';
import { ShelfPage } from './catalog/shelf';
import { CorePage } from './core/core';
import { EnginePage } from './core/engine';
import { VerdictPage } from './core/verdict';
import { HomePage } from './home/home';
import { InboxPage } from './inbox/inbox';
import { MachineFilesPage } from './machine-files/machine-files';
import { MachineHeader } from './machine/header';
import { MachineTab } from './machine/machine';
import { NewMachine } from './new-machine/new-machine';
import { AddPasswordSheet } from './passwords/add-sheet';
import { PasswordName } from './passwords/name';
import { PasswordsPage } from './passwords/passwords';
import { PlaceholderPage } from './placeholder';
import { StoragePage } from './storage/storage';

function Shelf({
    shelf,
    item,
    go,
    macs,
}: {
    shelf: 'apps' | 'settings' | 'skills';
    item: string | undefined;
    go: PageProps['go'];
    macs: PageProps['home']['macs'];
}) {
    return item === undefined ? (
        <ShelfPage go={go} shelf={shelf} />
    ) : (
        <ItemPage go={go} macs={macs} name={item} shelf={shelf} />
    );
}

function Files({
    file,
    go,
    newMachine,
}: {
    file: string | undefined;
    go: PageProps['go'];
    newMachine: PageProps['newMachine'];
}) {
    return <MachineFilesPage go={go} key={file ?? ''} onNewMachine={newMachine} opened={file} />;
}

function ItemName({ page, item }: { page: PlainPage; item: string }) {
    return page === 'passwords' ? <PasswordName id={item} /> : item;
}

function Head({ route, found, go, newMachine }: PageProps) {
    return route.page === 'machine' && found ? (
        <MachineHeader found={found} go={go} onNewMachine={newMachine} tab={route.tab} />
    ) : null;
}

function Body({ route, found, home, go, Gallery, newMachine, addMac }: PageProps) {
    switch (route.page) {
        case 'home': {
            return (
                <HomePage
                    go={go}
                    onAddMac={addMac}
                    onNewMachine={
                        newMachine === undefined
                            ? undefined
                            : () => {
                                  newMachine();
                              }
                    }
                    {...home}
                />
            );
        }
        case 'inbox': {
            return <InboxPage go={go} {...home} />;
        }
        case 'storage': {
            return <StoragePage go={go} macs={home.macs} />;
        }
        case 'engine': {
            return <EnginePage />;
        }
        case 'machine': {
            return found ? (
                <MachineTab found={found} go={go} tab={route.tab} />
            ) : (
                <PlaceholderPage title={route.machine} />
            );
        }
        case 'machine-files': {
            return <Files file={route.file} go={go} newMachine={newMachine} />;
        }
        case 'passwords': {
            return <PasswordsPage go={go} key={route.item ?? ''} opened={route.item} />;
        }
        case 'catalog-apps': {
            return <Shelf go={go} item={route.item} macs={home.macs} shelf="apps" />;
        }
        case 'catalog-skills': {
            return <Shelf go={go} item={route.item} macs={home.macs} shelf="skills" />;
        }
        case 'catalog-settings': {
            return <Shelf go={go} item={route.item} macs={home.macs} shelf="settings" />;
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

export const PAGES: Pages = {
    AddMac,
    AddPassword: AddPasswordSheet,
    Body,
    Confirm: GestureConfirm,
    Core: CorePage,
    Head,
    ItemName,
    NewMachine,
    routeOfNeed,
    Verdict: VerdictPage,
};
