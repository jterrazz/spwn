import { test } from '@jterrazz/test/config';
import { defineConfig } from '@jterrazz/typescript/config';

const SMOKE = process.env.SPWN_SMOKE !== undefined;

export default defineConfig({
    platform: 'node',
    presets: [test()],
    test: {
        projects: {
            cli: {
                exclude: SMOKE ? [] : ['cli/smoke/**'],
                include: ['cli/**/*.spec.ts'],
                literate: { include: ['cli/**/*.spec.yaml'], runner: './cli/cli.specification.ts' },
                reason: 'this member IS the specs tree, so its specifications sit at its root, where the derived globs (specs/cli/…) do not look; a docker-aware spec spawns real containers, two minutes at most; a smoke spec builds a world image from scratch and tags the one shared spwn/world:latest, so only `make test-smoke` (SPWN_SMOKE=1) runs them, one file at a time',
                serial: SMOKE,
                timeout: SMOKE ? 600_000 : 120_000,
            },
        },
    },
});
