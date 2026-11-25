import { waitUntil } from '@jterrazz/test';
import type { ReactElement } from 'react';

import type { OpenerPort } from '../../application/ports/opener.port';
import { findMachine } from '../../domain/machines/machines';
import { documentPresence } from '../../infrastructure/outbound/document-presence/document-presence';
import { Client } from '../../infrastructure/outbound/engine-link/client';
import { holdingRead } from '../../infrastructure/outbound/engine-replay/__tests__/held-read';
import type { HeldRead } from '../../infrastructure/outbound/engine-replay/__tests__/held-read';
import { stagedReplay } from '../../infrastructure/outbound/engine-replay/replay';
import type { Pace, Recording, Stage } from '../../infrastructure/outbound/engine-replay/replay';
import { NO_OPENER } from '../../infrastructure/outbound/opener-tauri/tauri-opener';
import { EngineProvider } from '../../presentation/dependencies/dependencies';
import type { Dependencies } from '../../presentation/dependencies/dependencies';
import type { Route } from '../../presentation/navigation/route';
import { RouterProvider, useRouter } from '../../presentation/navigation/router';
import { PageArea, useStation } from '../../presentation/navigation/window';
import { PAGES } from '../../presentation/screens/registry';
import { composeReplay } from '../compose';

const { Confirm } = PAGES;

function Area() {
    const { route, go } = useRouter();
    const { home, macs } = useStation();
    const found = route.page === 'machine' ? findMachine(macs, route.machine) : undefined;
    return (
        <main>
            <PageArea found={found} go={go} home={home} pages={PAGES} route={route} />
            <Confirm />
        </main>
    );
}

export function pageOver(recording: Recording, initial: Route, opener?: OpenerPort, pace?: Pace) {
    return (
        <EngineProvider dependencies={composeReplay({ opener, pace, recording })}>
            <RouterProvider initial={initial}>
                <Area />
            </RouterProvider>
        </EngineProvider>
    );
}

export type Staged = { dependencies: Dependencies; stage: Stage };

// value: a world whose exchanges the spec can count, so it knows when what it draws is drawn.
export function stagedOver(recording: Recording): Staged {
    const stage = stagedReplay(recording);
    return {
        dependencies: {
            engine: new Client(stage.transport),
            opener: NO_OPENER,
            presence: documentPresence(),
        },
        stage,
    };
}

export type HeldPage = { ui: ReactElement; held: HeldRead; stage: Stage };

// value: a page over a world whose reads of one name wait until released, the rest answering as recorded.
export function heldPageOver(recording: Recording, initial: Route, name: string): HeldPage {
    const stage = stagedReplay(recording);
    const held = holdingRead(stage, name);
    const dependencies = {
        engine: new Client(held.transport),
        opener: NO_OPENER,
        presence: documentPresence(),
    };
    const ui = (
        <EngineProvider dependencies={dependencies}>
            <RouterProvider initial={initial}>
                <Area />
            </RouterProvider>
        </EngineProvider>
    );
    return { held, stage, ui };
}

async function frame(): Promise<void> {
    await new Promise<void>((resolve) => {
        requestAnimationFrame(() => {
            resolve();
        });
    });
}

// value: resolves once every exchange the stage plays has ended and the page they answered is drawn.
export async function settled(stage: Stage): Promise<void> {
    await waitUntil(() => stage.open() === 0, { why: 'every exchange played' });
    await frame();
    await frame();
}

export function heightOf(selector: string): number {
    return document.querySelector(selector)?.getBoundingClientRect().height ?? 0;
}

// value: the gap the page's column draws between two of its pieces, as the styles loaded draw it.
export function gapAbove(title: string): number {
    const column = document.querySelector(`section[aria-label="${title}"]`)?.parentElement;
    return column ? Number.parseFloat(getComputedStyle(column).rowGap) || 0 : 0;
}

export function panelHeight(title: string): number {
    return heightOf(`section[aria-label="${title}"]`);
}
