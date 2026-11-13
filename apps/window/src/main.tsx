import './design/document.css';
import { invoke } from '@tauri-apps/api/core';
import { lazy, StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { Client } from './engine/client';
import { NO_OPENER, tauriOpener } from './engine/opener';
import { EngineProvider } from './engine/react';
import { replayTransport } from './engine/replay';
import { tauriTransport } from './engine/tauri';
import type { Transport } from './engine/transport';
import { PAGES } from './pages/registry';
import { Boundary } from './shell/boundary';
import { placeOf, wearDress } from './shell/chrome';
import { startRoute } from './shell/route';
import type { Route } from './shell/route';
import { recalled, remember } from './shell/route-memory';
import type { Memory } from './shell/route-memory';
import { Window } from './shell/window';

const proof = import.meta.env.MODE !== 'production';
const Gallery = proof ? lazy(async () => await import('./pages/gallery/gallery')) : undefined;
const query = new URLSearchParams(globalThis.location.search);
const inTauri = '__TAURI_INTERNALS__' in globalThis;

function memoryOf(): Memory | undefined {
    try {
        return globalThis.localStorage;
    } catch {
        return undefined;
    }
}

const memory = memoryOf();

function initialRoute(): Route {
    if (Gallery && query.has('gallery')) {
        return { page: 'gallery' };
    }
    if (proof && !inTauri) {
        return startRoute(query);
    }
    return recalled(memory, Gallery !== undefined) ?? { page: 'home' };
}

function onRoute(route: Route): void {
    remember(memory, route);
}

function absent(reason: string): Transport {
    return replayTransport(
        { exchanges: [], station: '', world: 'none' },
        { kind: 'absent', reason, retryIn: 30_000, socket: 'no socket' },
    );
}

async function transportOf(): Promise<Transport> {
    if (inTauri) {
        return tauriTransport();
    }
    const world = query.get('world') ?? 'two-macs';
    if (!proof || world === 'none') {
        return absent('no engine bridge outside the Tauri window');
    }
    const { recordingFor } = await import('./fixtures/recordings');
    const pace = { gestureFrame: Number(query.get('pace') ?? '0') };
    return replayTransport(await recordingFor(world), undefined, pace);
}

wearDress(
    document.documentElement,
    placeOf(inTauri, navigator.userAgent, proof && query.get('chrome') === 'overlay'),
    globalThis,
);

const root = document.querySelector('#root');
if (root) {
    const client = new Client(await transportOf());
    createRoot(root).render(
        <StrictMode>
            <Boundary
                onReload={() => {
                    globalThis.location.reload();
                }}
                where="the window"
            >
                <EngineProvider
                    client={client}
                    opener={inTauri ? tauriOpener({ invoke }) : NO_OPENER}
                >
                    <Window
                        Gallery={Gallery}
                        initial={initialRoute()}
                        onRoute={onRoute}
                        pages={PAGES}
                    />
                </EngineProvider>
            </Boundary>
        </StrictMode>,
    );
}
