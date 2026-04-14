import { existsSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, test } from 'vitest';

import { MindAssertion } from '../../setup/mind-assertion.js';
import {
    expectLine,
    expectNoLine,
    expectTableHeader,
    stripAnsi,
} from '../../setup/output-helpers.js';
import {
    createTestContext,
    parseWorldId,
    type TestContext,
} from '../../setup/spwn.specification.js';

/**
 * COMPLETE AGENT LIFECYCLE E2E TEST
 *
 * Tests the full user journey:
 *   create → spawn → inspect → destroy → dream
 *
 * Exercises the complete end-to-end flow end to end.
 */
describe('complete agent lifecycle', () => {
    let ctx: TestContext;

    afterEach(() => {
        ctx?.cleanup();
    });

    test('complete agent lifecycle: create → spawn → inspect → destroy → dream', () => {
        // ── STEP 1: Initialize universe ─────────────────────────────
        ctx = createTestContext();
        const initResult = ctx.spwn(['init']);
        expect(initResult.exitCode).toBe(0);

        // ── STEP 2: Verify default agent exists ─────────────────────
        // CreateTestContext already scaffolds a `neo` agent on disk.
        const agentLs = ctx.spwn(['agent', 'ls']);
        expect(agentLs.exitCode).toBe(0);
        expectTableHeader(agentLs.output, ['NAME', 'WORLD', 'STATUS']);
        expect(stripAnsi(agentLs.output)).toContain('neo');

        // ── STEP 3: Spawn world ─────────────────────────────────────
        const spawnResult = ctx.spwn(['up', '--agent', 'neo', '-w', ctx.home], 60_000);
        expect(spawnResult.exitCode).toBe(0);
        const worldId = parseWorldId(spawnResult.output);
        expect(worldId).toBeTruthy();
        expectLine(spawnResult.output, /✓ Created container\s+(?:spwn-world|w)-\w+-\d{5}/);

        // ── STEP 4: Verify world in ls ──────────────────────────────
        const worldLs = ctx.spwn(['ls']);
        expect(worldLs.exitCode).toBe(0);
        expect(stripAnsi(worldLs.output)).toContain(worldId!);
        expectTableHeader(worldLs.output, ['ID', 'CONFIG', 'AGENTS', 'STATUS']);

        // ── STEP 5: Inspect world ───────────────────────────────────
        const inspectResult = ctx.spwn(['world', 'inspect', worldId!]);
        expect(inspectResult.exitCode).toBe(0);
        expectLine(
            inspectResult.output,
            new RegExp(`World:\\s+${worldId!.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)}`),
        );
        expectLine(inspectResult.output, /Status:\s+(running|idle)/);

        // ── STEP 6: Destroy world ───────────────────────────────────
        const downResult = ctx.spwn(['down', worldId!], 30_000);
        expect(downResult.exitCode).toBe(0);
        expectLine(downResult.output, /✓ World destroyed\. Agent survives\./);

        // Verify world is gone from ls
        const lsAfterDown = ctx.spwn(['ls']);
        expect(lsAfterDown.exitCode).toBe(0);
        expectNoLine(
            lsAfterDown.output,
            new RegExp(worldId!.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`)),
        );

        // ── STEP 7: Dream → verify result ───────────────────────────
        const dreamResult = ctx.spwn(['agent', 'dream', 'neo']);
        expect(dreamResult.exitCode).toBe(0);
        expectLine(dreamResult.output, /→ Dreaming for agent "neo"\.\.\./);

        // ── STEP 8: Cleanup - remove agent ──────────────────────────
        const rmResult = ctx.spwn(['agent', 'rm', 'neo']);
        expect(rmResult.exitCode).toBe(0);
        expectLine(rmResult.output, /✓ Deleted agent\s+neo/);

        // Verify agent directory is gone
        expect(existsSync(join(ctx.home, 'agents', 'neo'))).toBe(false);
    });

    test('agent shows correct mind layout on disk', () => {
        // GIVEN - fresh context with the default neo agent
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - adding identity content
        const corePath = join(ctx.home, 'agents', 'neo', 'core', 'default.md');
        writeFileSync(
            corePath,
            '# Neo\nYou are a code architect specializing in distributed systems.\n',
        );

        // THEN - mind layers exist on disk
        new MindAssertion(ctx.home, 'neo').exists().hasLayer('core').hasFile('core/default.md');
    });

    test('multiple agents can coexist in the same universe', () => {
        // GIVEN - initialized universe with the default neo agent
        ctx = createTestContext();
        ctx.spwn(['init']);

        // WHEN - creating a second agent
        const newResult = ctx.spwn(['agent', 'new', 'trinity']);
        expect(newResult.exitCode).toBe(0);

        // THEN - both agents appear in agent ls
        const agentLs = ctx.spwn(['agent', 'ls']);
        expect(agentLs.exitCode).toBe(0);
        const output = stripAnsi(agentLs.output);
        expect(output).toContain('neo');
        expect(output).toContain('trinity');

        // CLEANUP
        ctx.spwn(['agent', 'rm', 'trinity']);
    });
});
