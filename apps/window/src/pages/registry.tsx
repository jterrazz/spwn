import { Suspense } from 'react';

import { TITLES } from '../shell/window';
import type { PageProps, Pages } from '../shell/window';
import { AddMac } from './add-mac/add-mac';
import { ItemPage } from './catalog/item';
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
    Body,
    Confirm: GestureConfirm,
    Head,
    NewMachine,
    routeOfNeed,
};
