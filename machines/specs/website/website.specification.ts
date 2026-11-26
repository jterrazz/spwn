import { specification } from '@jterrazz/test';
import { afterAll } from 'vitest';

// invariant: the site is the window's proof build as `make test` left it, its replay transport on, so a visit to
// `/?world=<world>&page=<page>` draws the shipped entry over that world's recorded answers
export const { cleanup, website } = await specification.website({
    server: {
        command:
            'cd ../apps/desktop && exec npx vite preview --outDir .artifacts/proof --host 127.0.0.1 --strictPort --port "$PORT"',
        ready: '/',
    },
});

afterAll(cleanup);

// value: the storage of a window whose column was pinned; a visit is 1280 px wide, where an unpinned column hides.
export const PINNED = { local: { 'station.window.column': 'pinned' } };
