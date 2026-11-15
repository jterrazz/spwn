import type { OpenerPort } from '../../application/ports/opener.port';
import { findMachine } from '../../domain/machines/machines';
import { EngineProvider } from '../../engine/react';
import type { Pace, Recording } from '../../infrastructure/outbound/engine-replay/replay';
import { PAGES } from '../../pages/registry';
import type { Route } from '../../shell/route';
import { RouterProvider, useRouter } from '../../shell/router';
import { PageArea, useStation } from '../../shell/window';
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
