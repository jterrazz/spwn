import './presentation/ui/document.css';
import { lazy, StrictMode } from 'react';
import { createRoot } from 'react-dom/client';

import { compose, composeGallerySample } from './composition/compose';
import { EngineProvider } from './presentation/dependencies/dependencies';
import { Boundary } from './presentation/navigation/boundary';
import { placeOf, wearDress } from './presentation/navigation/chrome';
import { startRoute } from './presentation/navigation/route';
import type { Route } from './presentation/navigation/route';
import { recalled, remember } from './presentation/navigation/route-memory';
import type { Memory } from './presentation/navigation/route-memory';
import { Window } from './presentation/navigation/window';
import { PAGES } from './presentation/screens/registry';

const proof = import.meta.env.MODE !== 'production';

async function galleryOf() {
    const { default: Drawn } = await import('./presentation/screens/gallery/gallery');
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
