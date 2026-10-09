import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

const SMOKE = process.env.SPWN_SMOKE !== undefined;

export default defineConfig({
    platform: 'node',
    presets: [test()],
    test: {
        projects: {
            cli: {
                exclude: SMOKE
                    ? []
                    : ['cli/init/real-build.spec.ts', 'cli/up/upgrade-cycle.spec.ts'],
                include: SMOKE
                    ? ['cli/init/real-build.spec.ts', 'cli/up/upgrade-cycle.spec.ts']
                    : ['cli/**/*.spec.ts'],
                literate: SMOKE
                    ? false
                    : { include: ['cli/**/*.spec.yaml'], runner: './cli/cli.specification.ts' },
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived globs (specs/cli/…) do not look; a docker-aware spec spawns real containers, two minutes at most; a smoke spec builds a world image from scratch and tags the one shared spwn/world:latest, so only `make test-smoke` (SPWN_SMOKE=1) runs them, one file at a time',
                serial: SMOKE,
                timeout: SMOKE ? 600_000 : 120_000,
            },
            website: {
                include: ['website/**/*.spec.ts'],
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived globs do not look; every file starts its own `spwn web` and `next dev`, and two dev servers of one app share its build folder, so the files run one at a time, and the first compile of a page outlasts the default budget',
                serial: true,
                timeout: 120_000,
            },
        },
    },
});
