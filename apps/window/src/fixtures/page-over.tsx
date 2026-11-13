import { Client } from '../engine/client';
import { NO_OPENER } from '../engine/opener';
import type { Opener } from '../engine/opener';
import { EngineProvider } from '../engine/react';
import { replayTransport } from '../engine/replay';
import type { Recording } from '../engine/replay';
import { findMachine } from '../model/machines';
import { PAGES } from '../pages/registry';
import type { Route } from '../shell/route';
import { RouterProvider, useRouter } from '../shell/router';
import { PageArea, useStation } from '../shell/window';

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

export function pageOver(recording: Recording, initial: Route, opener: Opener = NO_OPENER) {
    const client = new Client(replayTransport(recording));
    return (
        <EngineProvider client={client} opener={opener}>
            <RouterProvider initial={initial}>
                <Area />
            </RouterProvider>
        </EngineProvider>
    );
}
