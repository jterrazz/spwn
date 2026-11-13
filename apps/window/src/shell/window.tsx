import type { MachinesMacView, OverviewNeed, OverviewReadOutput } from '@jterrazz/station-api';
import { useCallback, useState } from 'react';
import type { ComponentType } from 'react';

import { Screen } from '../design/index';
import { engineViewStateOf } from '../engine/provisional';
import { useRead, useWatch } from '../engine/react';
import type { Reading } from '../engine/react';
import { findMachine, macsOf } from '../model/machines';
import type { Found } from '../model/machines';
import { Boundary } from './boundary';
import { Find } from './find';
import type { Action } from './finds';
import { onTabKey, useShortcuts } from './keys';
import { Lost, useLost } from './lost';
import { IN_CATALOG, machineRoute, TAB_PANEL } from './route';
import type { PlainPage, Route } from './route';
import { RouterProvider, useRouter } from './router';
import { Sidebar } from './sidebar';
import { Toolbar } from './toolbar';
import type { Crumb } from './toolbar';

import styles from './window.module.css';

export const TITLES: Record<PlainPage, string> = {
    home: 'Home',
    inbox: 'Inbox',
    storage: 'Storage',
    'machine-files': 'Machine files',
    catalog: 'Catalog',
    'catalog-apps': 'Apps',
    'catalog-skills': 'Skills',
    'catalog-settings': 'Settings',
    gallery: 'Gallery',
};

type Go = (to: Route) => void;

export type Station = {
    macs: readonly MachinesMacView[];
    overview: Reading<OverviewReadOutput>;
};

export type PageProps = {
    route: Route;
    found: Found | undefined;
    home: Station;
    go: Go;
    Gallery?: ComponentType | undefined;
    newMachine?: (() => void) | undefined;
};

export type Pages = {
    Head: ComponentType<PageProps>;
    Body: ComponentType<PageProps>;
    Confirm: ComponentType;
    NewMachine: ComponentType<{ go: Go; onClose: () => void }>;
    routeOfNeed: (need: OverviewNeed) => Route;
};

export type WindowProps = {
    initial: Route;
    pages: Pages;
    Gallery?: ComponentType | undefined;
    onRoute?: ((route: Route) => void) | undefined;
};

export function Window({ initial, pages, Gallery, onRoute }: WindowProps) {
    return (
        <RouterProvider initial={initial} onRoute={onRoute}>
            <Frame Gallery={Gallery} pages={pages} />
        </RouterProvider>
    );
}

type Sheet = 'find' | 'new-machine' | 'none';

function crumbsOf(route: Route, found: Found | undefined, go: Go): Crumb[] {
    if (route.page !== 'machine') {
        const page = { label: route.item ?? TITLES[route.page] };
        if (!IN_CATALOG.has(route.page)) {
            return [page];
        }
        const catalog = {
            label: TITLES.catalog,
            open: () => {
                go({ page: 'catalog' });
            },
        };
        return [catalog, page];
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

export function whereOf(route: Route): string {
    if (route.page === 'machine') {
        return `${route.machine} › ${route.tab}`;
    }
    return route.item === undefined ? TITLES[route.page] : `${TITLES.catalog} › ${route.item}`;
}

type AreaProps = PageProps & { pages: Pages; panel?: boolean };

export function PageArea({ pages, panel = false, ...props }: AreaProps) {
    const { route } = props;
    const where = whereOf(route);
    const { Head, Body } = pages;
    return (
        <>
            <div className={styles.head} onKeyDown={onTabKey} role="none">
                <Boundary key={`head ${where}`} where={`the header of ${where}`}>
                    <Head {...props} />
                </Boundary>
            </div>
            <div
                aria-label={panel && route.page === 'machine' ? route.tab : undefined}
                className={styles.scroll}
                id={panel && route.page === 'machine' ? TAB_PANEL : undefined}
                key={where}
                role={panel && route.page === 'machine' ? 'tabpanel' : undefined}
            >
                <Boundary where={where}>
                    <Body {...props} />
                </Boundary>
            </div>
        </>
    );
}

export function useStation() {
    const machines = useWatch('machines.list', {});
    const overview = useWatch('overview.read', {});
    const engine = useRead('engine.read', {});
    const macs = macsOf(machines.kind === 'read' ? machines.out : undefined);
    const needs = overview.kind === 'read' ? (overview.out.Summary.Needs ?? []) : [];
    const home: Station = { macs, overview };
    const listed = machines.kind === 'read' && machines.done;
    return { home, listed, macs, needs, rail: { engine, macs, needs } };
}

type SheetsProps = {
    sheet: Sheet;
    onClose: () => void;
    act: (action: Action) => void;
    go: Go;
    pages: Pages;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    withGallery: boolean;
};

function Sheets({ sheet, onClose, act, go, pages, macs, needs, withGallery }: SheetsProps) {
    if (sheet === 'find') {
        return (
            <Find
                act={act}
                go={go}
                macs={macs}
                needs={needs}
                onClose={onClose}
                routeOfNeed={pages.routeOfNeed}
                titles={TITLES}
                withGallery={withGallery}
            />
        );
    }
    const { NewMachine } = pages;
    return sheet === 'new-machine' ? <NewMachine go={go} onClose={onClose} /> : null;
}

function useSheets() {
    const [sidebar, setSidebar] = useState(true);
    const [sheet, setSheet] = useState<Sheet>('none');
    const find = useCallback(() => {
        setSheet('find');
    }, []);
    const close = useCallback(() => {
        setSheet('none');
    }, []);
    const act = (action: Action) => {
        if (action === 'new-machine') {
            setSheet('new-machine');
        } else {
            setSidebar((open) => !open);
        }
    };
    return { act, close, find, sheet, sidebar };
}

// value: while this Mac's engine is stopped, Start the engine leads Home and New machine stands back.
function engineStopped(macs: readonly MachinesMacView[]): boolean {
    const thisMac = macs.find((mac) => mac.ThisMac);
    return thisMac !== undefined && engineViewStateOf(thisMac.Engine) === 'stopped';
}

function Frame({ Gallery, pages }: { Gallery?: ComponentType | undefined; pages: Pages }) {
    const router = useRouter();
    const { route, go, back, canGoBack } = router;
    const { home, listed, macs, needs, rail } = useStation();
    const { act, close, find, sheet, sidebar } = useSheets();
    useShortcuts(router, find, Gallery !== undefined);
    const { lost, dismiss } = useLost(router, macs, listed);
    const found = route.page === 'machine' ? findMachine(macs, route.machine) : undefined;
    const newMachine = () => {
        act('new-machine');
    };
    return (
        <Screen>
            <div className={`${styles.window} ${sidebar ? '' : styles.collapsed}`}>
                {sidebar ? <Sidebar {...rail} go={go} onFind={find} route={route} /> : null}
                <main className={styles.main}>
                    <Toolbar
                        canGoBack={canGoBack}
                        crumbs={crumbsOf(route, found, go)}
                        newMachineLeads={!engineStopped(macs)}
                        onBack={back}
                        onFind={find}
                        onNewMachine={route.page === 'home' ? newMachine : undefined}
                        onToggleSidebar={() => {
                            act('sidebar');
                        }}
                        sidebar={sidebar}
                    />
                    {lost === undefined ? null : <Lost machine={lost} onDismiss={dismiss} />}
                    <PageArea
                        found={found}
                        Gallery={Gallery}
                        go={go}
                        home={home}
                        newMachine={newMachine}
                        pages={pages}
                        panel
                        route={route}
                    />
                </main>
            </div>
            <Sheets
                act={act}
                go={go}
                macs={macs}
                needs={needs}
                onClose={close}
                pages={pages}
                sheet={sheet}
                withGallery={Gallery !== undefined}
            />
            <pages.Confirm />
        </Screen>
    );
}
