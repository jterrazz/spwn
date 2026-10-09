import { directory } from '@jterrazz/test';
import { describe, expect, test } from 'vitest';

import { cli } from '../cli.specification.js';
import { labelsOf } from './__setup__/world.js';

// hazard: a document cannot compare two sessions nor read their containers back

const real = cli.with({ env: { SPWN_BASE_IMAGE: null } });

const INSIDE = [
    'command -v python3 git qmd >/dev/null && echo tools',
    'cat /agents/neo/playbooks/arithmetic.md',
    'cat /agents/neo/SOUL.md',
    'cat /agents/trinity/SOUL.md',
].join(' && ');

const imageVersionOf = (inspected: unknown): string | undefined =>
    labelsOf(inspected)['sh.spwn.image-version'];

describe('upgrade-cycle', () => {
    test('a project that gains a tool, a playbook, a soul and an agent rebuilds and carries every change', async () => {
        // Given - the scaffold brought up on a real build, then the same project grown by the delta and brought up again
        await using baseline = await real
            .fixture(directory('./__fixtures__/scaffolded-project/'))
            .exec('up');
        await using upgraded = await real
            .fixture(directory('./__fixtures__/scaffolded-project/'))
            .fixture(directory('./__fixtures__/upgrade-delta/'))
            .exec('up');

        // Then - a new image version, and inside the world the new tool, playbook, soul and second home
        const inside = await upgraded.container('neo').exec(`sh -c '${INSIDE}'`);
        expect({
            exitCodes: [baseline.exitCode, upgraded.exitCode],
            inside: inside.stdout.text,
            rebuilt:
                imageVersionOf(baseline.container('neo').inspect.value) !==
                imageVersionOf(upgraded.container('neo').inspect.value),
        }).toStrictEqual({
            exitCodes: [0, 0],
            inside: 'tools\n# Arithmetic\n\nReturn 2 + 2 when asked.\n# Upgraded identity\n\nI am the post-upgrade neo.\n# trinity\n\nOperator, partner, edge case specialist.\n',
            rebuilt: true,
        });
    });
});
