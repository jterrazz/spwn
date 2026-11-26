import type {
    CoreGuideGuide,
    MachinesMacView,
    OverviewNeed,
    OverviewReadOutput,
} from '@jterrazz/station-api';
import { useMemo, useSyncExternalStore } from 'react';
import type { ComponentType } from 'react';

import { findMachine, macsOf } from '../../domain/machines/machines';
import type { Found } from '../../domain/machines/machines';
import { useRead, useWatch } from '../dependencies/dependencies';
import type { Reading } from '../dependencies/dependencies';
import { Screen } from '../ui/index';
import { rollUpState } from '../words/shown';
import { Boundary } from './boundary';
import { useCarried } from './carried';
import { ColumnSlot } from './column';
import { ColumnState } from './column-state';
import { Find } from './find';
import { onTabKey } from './keys';
import { Lost, useLost } from './lost';
import { NoEngine, useHeld } from './no-engine';
import type { Held } from './no-engine';
import { Opened } from './opened';
import type { Opening } from './opened';
import { PathBar } from './path-bar';
import { TAB_PANEL } from './route';
import type { PlainPage, Route } from './route';
import type { Memory } from './route-memory';
import { RouterProvider, useRouter } from './router';
import { useSettledRoute } from './settled';
import { isGlobal } from './spaces';
import { TITLES } from './titles';
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
    guide?: ((act: string) => void) | undefined;
};

export type { Opening } from './opened';
export type Pages = {
    Head: ComponentType<PageProps>;
    Body: ComponentType<PageProps>;
    Confirm: ComponentType;
    Guide: ComponentType<{
        act: string;
        go: Go;
        onClose: () => void;
        served: CoreGuideGuide | undefined;
    }>;
    Core: ComponentType;
    NewMachine: ComponentType<{
        go: Go;
        onClose: () => void;
        onGuide: (act: string, served: CoreGuideGuide | undefined) => void;
        opening: Opening;
    }>;
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
    memory?: Memory | undefined;
};

export function Window({ initial, pages, Gallery, onRoute, memory }: WindowProps) {
    return (
        <RouterProvider initial={initial} onRoute={onRoute}>
            <Frame Gallery={Gallery} memory={memory} pages={pages} />
        </RouterProvider>
    );
}

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
    const rollUp = machines.kind === 'read' ? machines.out.All.Standing : undefined;
    const standing = rollUp === undefined ? undefined : rollUpState(rollUp);
    return { drawn, engine, home, listed, macs, needs, side: { carried, macs }, standing };
}

type SheetsProps = {
    opened: Opened;
    go: Go;
    pages: Pages;
    macs: readonly MachinesMacView[];
    needs: readonly OverviewNeed[];
    withGallery: boolean;
};

function Sheets({ opened, go, pages, macs, needs, withGallery }: SheetsProps) {
    const sheet = useSyncExternalStore(opened.subscribe, opened.shown);
    const { NewMachine, AddMac, AddPassword, Guide } = pages;
    const { addMac: onAddMac, close: onClose, guide: onGuide } = opened;
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
            return (
                <NewMachine go={go} onClose={onClose} onGuide={onGuide} opening={sheet.opening} />
            );
        }
        case 'guide': {
            return <Guide act={sheet.act} go={go} onClose={onClose} served={sheet.served} />;
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
    const opened = useMemo(() => new Opened(), []);
    return { opened };
}

type FrameProps = {
    Gallery?: ComponentType | undefined;
    pages: Pages;
    memory: Memory | undefined;
};

// invariant: a space opens where this window last was in it; only Machine opens today, Agent and OS wait.
const NOWHERE = () => undefined;

function Content({ area, held }: { area: AreaProps; held: Held | undefined }) {
    if (held === undefined) {
        return <PageArea {...area} panel />;
    }
    return <NoEngine Core={held === 'verdict' ? area.pages.Verdict : area.pages.Core} />;
}

type MainProps = {
    area: AreaProps;
    held: Held | undefined;
    lost: ReturnType<typeof useLost>;
};

function Main({ area, held, lost: { lost, dismiss } }: MainProps) {
    return (
        <main className={styles.main}>
            {lost === undefined || held !== undefined ? null : (
                <Lost machine={lost} onDismiss={dismiss} />
            )}
            <Content area={area} held={held} />
        </main>
    );
}

// invariant: one tree whatever the link says, so an absence, a verdict and a relink remount no page and close no
// invariant: sheet; the core's panel or the verdict's stands in the page's place.
function Frame({ Gallery, pages, memory }: FrameProps) {
    const router = useRouter();
    const { route, go } = router;
    const { drawn, engine, home, listed, macs, needs, side, standing } = useStation();
    const held = useHeld(drawn);
    const quiet = held !== undefined;
    const sheets = useSheets();
    const column = useMemo(() => new ColumnState(memory, globalThis.innerWidth), [memory]);
    const { opened } = sheets;
    const { addMac, guide, newMachine } = opened;
    const withGallery = Gallery !== undefined;
    const spaced = quiet || !isGlobal(route);
    const onSpace = useSpaceKeys(
        router,
        macs,
        { onColumn: spaced ? column.toggle : NOWHERE, onFind: opened.find },
        { on: !quiet, withGallery },
    );
    const lost = useLost(router, macs, listed);
    const found = route.page === 'machine' ? findMachine(macs, route.machine) : undefined;
    useSettledRoute(router, found);
    const area = { addMac, found, Gallery, go, guide, home, newMachine, pages, route };
    const aside = { ...side, engine, go, needs, quiet, route, routeOfNeed: pages.routeOfNeed };
    return (
        <Screen>
            <div className={styles.window}>
                <PathBar
                    column={spaced ? column : undefined}
                    engine={engine}
                    found={found}
                    ItemName={pages.ItemName}
                    macs={macs}
                    needs={needs}
                    onAddPassword={opened.addPassword}
                    onFind={quiet ? NOWHERE : opened.find}
                    onNewMachine={opened.newBlank}
                    onSpace={onSpace}
                    quiet={quiet}
                    router={router}
                    standing={standing}
                />
                {spaced ? <ColumnSlot {...aside} column={column} /> : null}
                <Main area={area} held={held} lost={lost} />
            </div>
            {quiet ? null : (
                <Sheets
                    go={go}
                    macs={macs}
                    needs={needs}
                    opened={opened}
                    pages={pages}
                    withGallery={withGallery}
                />
            )}
            <pages.Confirm />
        </Screen>
    );
}
