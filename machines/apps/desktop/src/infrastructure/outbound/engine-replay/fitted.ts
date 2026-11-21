import type { Link } from '../../../application/ports/engine.port';
import { PAGE_REVISION } from '../../../domain/fit';
import { helloOf } from './hello';
import type { Recording } from './replay';

const PAGES: Readonly<Record<string, number>> = {
    'app-older': PAGE_REVISION + 1,
    'engine-older': PAGE_REVISION - 1,
};

// value: `?fit=engine-older` or `app-older`, the world's engine answering its hello one revision below the one the
// value: window draws, or one above it; any other value, or none, leaves its hello as recorded.
export function fittedLink(recording: Recording, fit: null | string): Link | undefined {
    const page = fit === null ? undefined : PAGES[fit];
    if (page === undefined) {
        return undefined;
    }
    const hello = { ...helloOf(recording), page };
    return { hello, kind: 'open', refused: undefined, socket: `replay of ${recording.world}` };
}
