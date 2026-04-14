import { afterEach, describe, expect, test } from 'vitest';

import { createAgent } from '../../setup/helpers.js';
import { expectLine, expectNoLine } from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

describe('colony multi-agent', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('spawn multi-agent world (first --agent is chief)', () => {
        // GIVEN - two agents
        ctx = createTestContext();
        createAgent(ctx.home, 'morpheus');
        ctx.spwn(['init']);

        // WHEN - spawning with two --agent flags
        const spawnResult = ctx.spwn(
            ['world', '--agent', 'morpheus', '--agent', 'neo', '-w', ctx.home],
            60_000,
        );

        expect(spawnResult.exitCode).toBe(0);
        expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);
        expectLine(spawnResult.output, /✓ Colony spawned\s+2 agent\(s\)/);

        const id = parseWorldId(spawnResult.output)!;
        ctx.world(id)
            .toBeRunning()
            .toHaveFile('/world/physics.md')
            .toHaveFile('/world/faculties.md');

        ctx.mind('neo').exists();
        ctx.mind('morpheus').exists();
    });

    test('destroying multi-agent world cleans up', () => {
        ctx = createTestContext();
        createAgent(ctx.home, 'morpheus');
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(
            ['world', '--agent', 'morpheus', '--agent', 'neo', '-w', ctx.home],
            60_000,
        );
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();

        ctx.world(id).toBeRunning();

        const destroyResult = ctx.spwn(['world', 'destroy', id], 30_000);

        expect(destroyResult.exitCode).toBe(0);
        expectLine(destroyResult.output, /✓ World destroyed\. Agent survives\./);

        ctx.world(id).toNotExist();

        const listResult = ctx.spwn(['world', 'list']);
        expectNoLine(
            listResult.output,
            new RegExp(id.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)),
        );

        ctx.state().noWorld(id);
    });
});
