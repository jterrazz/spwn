import './design/document.css';
import { lazy, StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { compose, composeGallerySample } from './composition/compose';
import { EngineProvider } from './engine/react';
import { PAGES } from './pages/registry';
import { Boundary } from './shell/boundary';
import { placeOf, wearDress } from './shell/chrome';
import { startRoute } from './shell/route';
import type { Route } from './shell/route';
import { recalled, remember } from './shell/route-memory';
import type { Memory } from './shell/route-memory';
import { Window } from './shell/window';

const proof = import.meta.env.MODE !== 'production';

async function galleryOf() {
    const { default: Drawn } = await import('./pages/gallery/gallery');
    const sample = composeGallerySample();
    function Sampled() {
        return <Drawn sample={sample} />;
    }
    return { default: Sampled };
}

const Gallery = proof ? lazy(galleryOf) : undefined;
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

wearDress(
    document.documentElement,
    placeOf(inTauri, navigator.userAgent, proof && query.get('chrome') === 'overlay'),
    globalThis,
);

const root = document.querySelector('#root');
if (root) {
    const dependencies = await compose({ inTauri, query });
    createRoot(root).render(
        <StrictMode>
            <Boundary
                onReload={() => {
                    globalThis.location.reload();
                }}
                where="the window"
            >
                <EngineProvider dependencies={dependencies}>
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
