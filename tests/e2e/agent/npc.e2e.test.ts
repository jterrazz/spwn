import { afterEach, describe, expect, test } from 'vitest';

import { stripAnsi } from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

describe('agent --ephemeral', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('ephemeral without --world flag fails', () => {
        // GIVEN - an initialized SPWN_HOME
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - running agent --ephemeral without --world flag
        const result = ctx.spwn(['agent', '--ephemeral', 'do something']);

        // THEN - exits with error about required world flag
        expect(result.exitCode).not.toBe(0);
        // AND - output indicates the --world flag is required (no stack traces)
        expect(stripAnsi(result.output)).not.toContain('TypeError');
        expect(stripAnsi(result.output)).not.toContain('ReferenceError');
    });

    test('ephemeral dispatches task in world', () => {
        // GIVEN - a running world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;
        expect(id).toBeTruthy();

        // Verify container is running before dispatching ephemeral
        ctx.world(id).toBeRunning();

        // WHEN - dispatching an ephemeral task
        const npcResult = ctx.spwn(
            ['agent', '--ephemeral', 'lint the code', '--world', id],
            30_000,
        );

        // THEN - succeeds and produces output
        expect(npcResult.exitCode).toBe(0);
        expect(npcResult.output.length).toBeGreaterThan(0);

        // AND - container is still running after ephemeral
        ctx.world(id).toBeRunning();
    });

    test('ephemeral does not create Mind directory', () => {
        // GIVEN - a running world
        ctx = createTestContext();
        ctx.spwn(['init']);
        const spawnResult = ctx.spwn(['world', '--agent', 'neo', '-w', ctx.home], 60_000);
        const id = parseWorldId(spawnResult.output)!;

        // WHEN - dispatching an ephemeral task
        ctx.spwn(['agent', '--ephemeral', 'check health', '--world', id]);

        // THEN - no ephemeral agent should appear in agent ls
        const list = ctx.spwn(['agent', 'ls']);
        expect(stripAnsi(list.output)).not.toContain('npc');
    });
});
