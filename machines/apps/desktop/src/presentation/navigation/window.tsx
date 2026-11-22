import type { MachinesMacView, OverviewNeed, OverviewReadOutput } from '@jterrazz/station-api';
import { useCallback, useState } from 'react';
import type { ComponentType } from 'react';

import { findMachine, macsOf } from '../../domain/machines/machines';
import type { Found } from '../../domain/machines/machines';
import { useRead, useWatch } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { Screen } from '../ui/index';
import { Boundary } from './boundary';
import { useCarried } from './carried';
import { TITLES } from './crumbs';
import { Find } from './find';
import { FrameToolbar } from './frame-toolbar';
import { onTabKey } from './keys';
import { Lost, useLost } from './lost';
import { NoEngine, useHeld } from './no-engine';
import type { Held } from './no-engine';
import { Rail } from './rail';
import { TAB_PANEL } from './route';
import type { PlainPage, Route } from './route';
import { RouterProvider, useRouter } from './router';
import { Sidebar } from './sidebar';
import { useSpaceKeys } from './use-spaces';

import styles from './window.module.css';

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
    newMachine?: ((opening?: Opening) => void) | undefined;
    addMac?: (() => void) | undefined;
};

// value: where New machine was opened from: a Mac's page names that Mac, a machine file's page that file.
export type Opening = { on?: string | undefined; from?: string | undefined };

export type Pages = {
    Head: ComponentType<PageProps>;
    Body: ComponentType<PageProps>;
    Confirm: ComponentType;
    Core: ComponentType;
    NewMachine: ComponentType<{ go: Go; onClose: () => void; opening: Opening }>;
    AddMac: ComponentType<{ go: Go; onClose: () => void }>;
    AddPassword: ComponentType<{ onClose: () => void }>;
    ItemName: ComponentType<{ page: PlainPage; item: string }>;
    routeOfNeed: (need: OverviewNeed) => Route;
    Verdict: ComponentType;
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

type Sheet =
    | { kind: 'add-mac' }
    | { kind: 'add-password' }
    | { kind: 'find' }
    | { kind: 'new-machine'; opening: Opening }
    | { kind: 'none' };

export function whereOf(route: Route): string {
    if (route.page === 'machine') {
        return `${route.machine} › ${route.tab}`;
    }
    return route.item === undefined ? TITLES[route.page] : `${TITLES[route.page]} › ${route.item}`;
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
    const overview = useWatch('overview.read', { Elsewhere: true });
    const engine = useRead('engine.read', {});
    const carried = useCarried();
    const macs = macsOf(machines.kind === 'read' ? machines.out : undefined);
    const needs = overview.kind === 'read' ? (overview.out.Summary.Needs ?? []) : [];
    const home: Station = { macs, overview };
    const drawn = machines.kind === 'read';
    const listed = drawn && machines.done;
    return { drawn, engine, home, listed, macs, needs, side: { carried, macs } };
}

type SheetsProps = {
    sheet: Sheet;
    onClose: () => void;
    onAddMac: () => void;
    go: Go;
    pages: Pages;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    withGallery: boolean;
};

function Sheets({ sheet, onClose, onAddMac, go, pages, macs, needs, withGallery }: SheetsProps) {
    const { NewMachine, AddMac, AddPassword } = pages;
    switch (sheet.kind) {
        case 'find': {
            return (
                <Find
                    go={go}
                    macs={macs}
                    needs={needs}
                    onAddMac={onAddMac}
                    onClose={onClose}
                    routeOfNeed={pages.routeOfNeed}
                    titles={TITLES}
                    withGallery={withGallery}
                />
            );
        }
        case 'new-machine': {
            return <NewMachine go={go} onClose={onClose} opening={sheet.opening} />;
        }
        case 'add-mac': {
            return <AddMac go={go} onClose={onClose} />;
        }
        case 'add-password': {
            return <AddPassword onClose={onClose} />;
        }
        case 'none': {
            return null;
        }
    }
}

function useSheets() {
    const [sidebar, setSidebar] = useState(true);
    const [sheet, setSheet] = useState<Sheet>({ kind: 'none' });
    const find = useCallback(() => {
        setSheet({ kind: 'find' });
    }, []);
    const close = useCallback(() => {
        setSheet({ kind: 'none' });
    }, []);
    const newMachine = useCallback((opening: Opening = {}) => {
        setSheet({ kind: 'new-machine', opening });
    }, []);
    const addMac = useCallback(() => {
        setSheet({ kind: 'add-mac' });
    }, []);
    const addPassword = useCallback(() => {
        setSheet({ kind: 'add-password' });
    }, []);
    const toggleSidebar = useCallback(() => {
        setSidebar((open) => !open);
    }, []);
    return { addMac, addPassword, close, find, newMachine, sheet, sidebar, toggleSidebar };
}

type FrameProps = { Gallery?: ComponentType | undefined; pages: Pages };

// invariant: a space opens where this window last was in it; only Machine opens today, Agent and OS wait.
const NOWHERE = () => undefined;

function Content({ area, held }: { area: AreaProps; held: Held | undefined }) {
    if (held === undefined) {
        return <PageArea {...area} panel />;
    }
    return <NoEngine Core={held === 'verdict' ? area.pages.Verdict : area.pages.Core} />;
}

// invariant: one tree whatever the link says, so an absence, a verdict and a relink remount no page and close no
// invariant: sheet; the core's panel or the verdict's stands in the page's place.
function Frame({ Gallery, pages }: FrameProps) {
    const router = useRouter();
    const { route, go } = router;
    const { drawn, engine, home, listed, macs, needs, side } = useStation();
    const held = useHeld(drawn);
    const quiet = held !== undefined;
    const { addMac, addPassword, close, find, newMachine, sheet, sidebar, toggleSidebar } =
        useSheets();
    const onSpace = useSpaceKeys(router, macs, find, {
        on: !quiet,
        withGallery: Gallery !== undefined,
    });
    const { lost, dismiss } = useLost(router, macs, listed);
    const found = route.page === 'machine' ? findMachine(macs, route.machine) : undefined;
    const area = { addMac, found, Gallery, go, home, newMachine, pages, route };
    const rail = { engine, go, needs, onSpace, quiet, route };
    const aside = { ...side, go, onFind: quiet ? NOWHERE : find, quiet, route };
    return (
        <Screen>
            <div className={`${styles.window} ${sidebar ? '' : styles.collapsed}`}>
                <Rail {...rail} />
                {sidebar ? <Sidebar {...aside} /> : null}
                <main className={styles.main}>
                    <FrameToolbar
                        found={found}
                        ItemName={pages.ItemName}
                        macs={macs}
                        onAddPassword={addPassword}
                        onNewMachine={() => {
                            newMachine();
                        }}
                        onToggleSidebar={toggleSidebar}
                        quiet={quiet}
                        router={router}
                    />
                    {lost === undefined || quiet ? null : (
                        <Lost machine={lost} onDismiss={dismiss} />
                    )}
                    <Content area={area} held={held} />
                </main>
            </div>
            {quiet ? null : (
                <Sheets
                    go={go}
                    macs={macs}
                    needs={needs}
                    onAddMac={addMac}
                    onClose={close}
                    pages={pages}
                    sheet={sheet}
                    withGallery={Gallery !== undefined}
                />
            )}
            <pages.Confirm />
        </Screen>
    );
}
