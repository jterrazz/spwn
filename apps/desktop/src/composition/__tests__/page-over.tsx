import type { OpenerPort } from '../../application/ports/opener.port';
import { findMachine } from '../../domain/machines/machines';
import type { Pace, Recording } from '../../infrastructure/outbound/engine-replay/replay';
import { PAGES } from '../../pages/registry';
import { EngineProvider } from '../../presentation/dependencies/dependencies';
import type { Route } from '../../presentation/navigation/route';
import { RouterProvider, useRouter } from '../../presentation/navigation/router';
import { PageArea, useStation } from '../../presentation/navigation/window';
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
