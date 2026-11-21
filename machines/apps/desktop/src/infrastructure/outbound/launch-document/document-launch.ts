import type { LaunchPort } from '../../../application/ports/launch.port';

// invariant: the faces `src/presentation/ui/document.css` imports, each loaded as soon as the script runs, so their
// invariant: files are fetched before the frame lays out its first text.
export const FACES = [
    '400 1em Geist',
    '500 1em Geist',
    '600 1em Geist',
    '400 1em "Geist Mono"',
    '500 1em "Geist Mono"',
] as const;

async function loaded(): Promise<void> {
    if (typeof document === 'undefined') {
        return;
    }
    await Promise.allSettled(FACES.map(async (face) => await document.fonts.load(face)));
}

export function documentLaunch(show: () => Promise<void>): LaunchPort {
    return {
        after: (ms, run) => {
            setTimeout(run, ms);
        },
        faces: loaded,
        show,
        since: () => performance.now(),
    };
}
