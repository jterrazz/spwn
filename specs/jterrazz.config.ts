import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

const SMOKE = process.env.SPWN_SMOKE !== undefined;

export default defineConfig({
    platform: 'node',
    presets: [test()],
    typescript: {
        knip: {
            entry: {
                'website/__setup__/project.ts':
                    'the throwaway project the server command of the website specification runs with node before `spwn web`',
            },
        },
        roots: {
            __setup__:
                'what the cli and website specifications share: the lock that gives one spec at a time the architect container the product names once',
            cli: 'the cli specification: this member IS the specs tree, so its specification folders sit at its root',
            website:
                'the website specification: this member IS the specs tree, so its specification folders sit at its root',
        },
    },
    test: {
        projects: {
            cli: {
                exclude: SMOKE ? [] : ['cli/up/real-build.spec.ts', 'cli/up/upgrade-cycle.spec.ts'],
                include: SMOKE
                    ? ['cli/up/real-build.spec.ts', 'cli/up/upgrade-cycle.spec.ts']
                    : ['cli/**/*.spec.ts'],
                literate: SMOKE
                    ? false
                    : { include: ['cli/**/*.spec.yaml'], runner: './cli/cli.specification.ts' },
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived globs (specs/cli/...) do not look; a docker-aware spec spawns real containers, two minutes at most; a smoke spec builds a world image from scratch and tags the one shared spwn/world:latest, so only `make test-smoke` (SPWN_SMOKE=1) runs them, one file at a time',
                serial: SMOKE,
                timeout: SMOKE ? 600_000 : 120_000,
            },
            website: {
                include: ['website/**/*.spec.ts'],
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived globs do not look; the run builds and serves the UI once, with the one API address its client is built with, and every file starts its own `spwn web` on that address, so the files run one at a time',
                serial: true,
                setup: ['website/__setup__/ui.global-setup.ts'],
            },
        },
    },
});
